import json
import unittest
from pathlib import Path
from tempfile import TemporaryDirectory

from stockquant.protocol import (
    ProtocolSchemaError,
    decode_request,
    decode_result,
    encode_request,
    encode_result,
)


ROOT = Path(__file__).resolve().parents[2]


class ProtocolTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.request_bytes = (ROOT / "contracts/examples/request.json").read_bytes()
        cls.result_bytes = (ROOT / "contracts/examples/result.json").read_bytes()

    def test_examples_decode_to_dtos(self):
        request = decode_request(self.request_bytes)
        result = decode_result(self.result_bytes)

        self.assertEqual(request.schema_version, "v1")
        self.assertEqual(request.mode, "raw_factors")
        self.assertEqual(request.data_ref.sha256, "c" * 64)
        self.assertEqual(result.status, "success")
        self.assertEqual(len(result.results), 1)
        self.assertEqual(result.results[0].ts_code, "000001.SZ")

    def test_request_rejects_invalid_contracts(self):
        cases = (
            ("missing request_id", lambda value: value.pop("request_id")),
            ("changed mode", lambda value: value.update(mode="final_score")),
            ("bad hash", lambda value: value.update(snapshot_hash="abc123")),
            ("invalid calendar date", lambda value: value.update(as_of="2026-02-30")),
            ("year zero date", lambda value: value.update(as_of="0000-01-01")),
            ("unknown field", lambda value: value.update(extra_field=True)),
        )
        original = json.loads(self.request_bytes)
        for name, change in cases:
            with self.subTest(name=name):
                payload = dict(original)
                change(payload)
                with self.assertRaises(ValueError):
                    decode_request(json.dumps(payload).encode())

    def test_field_order_does_not_change_request(self):
        original = json.loads(self.request_bytes)
        reordered = dict(reversed(list(original.items())))
        request = decode_request(json.dumps(reordered).encode())
        self.assertEqual(request.request_id, original["request_id"])

    def test_request_encoding_is_schema_validated(self):
        request = decode_request(self.request_bytes)
        self.assertEqual(decode_request(encode_request(request)), request)

    def test_result_rejects_multiple_json_documents(self):
        with self.assertRaises(ValueError):
            decode_result(self.result_bytes + b"\n{}")

    def test_result_rejects_numbers_outside_finite_float_range(self):
        payload = self.result_bytes.replace(b"momentum_60\":0.1", b"momentum_60\":1e9999")
        with self.assertRaises(ValueError):
            decode_result(payload)

    def test_large_factor_integer_matches_go_float64_semantics(self):
        payload = self.result_bytes.replace(b"momentum_60\":0.1", b"momentum_60\":9007199254740993")
        result = decode_result(payload)
        self.assertEqual(result.results[0].momentum_60, float(9007199254740993))

    def test_result_rejects_year_zero_date(self):
        document = json.loads(self.result_bytes)
        document["as_of"] = "0000-01-01"
        with self.assertRaises(ValueError):
            decode_result(json.dumps(document).encode())

    def test_result_rejects_invalid_error_array(self):
        document = json.loads(self.result_bytes)
        document["errors"] = {"code": "BAD_DATA"}
        with self.assertRaises(ValueError):
            decode_result(json.dumps(document).encode())

    def test_result_encoding_is_schema_validated(self):
        result = decode_result(self.result_bytes)
        encoded = encode_result(result)
        self.assertEqual(decode_result(encoded), result)

    def test_missing_schema_resource_blocks_decoding(self):
        with TemporaryDirectory() as directory:
            with self.assertRaises(ProtocolSchemaError):
                decode_request(self.request_bytes, schema_dir=Path(directory))


if __name__ == "__main__":
    unittest.main()
