"""与 Go 共用 JSON Schema v1 的严格编解码器。"""

from __future__ import annotations

import json
import math
from dataclasses import asdict, dataclass, field
from functools import lru_cache
from pathlib import Path
from typing import Any

from jsonschema import Draft202012Validator, FormatChecker
from jsonschema.exceptions import SchemaError


class ProtocolValidationError(ValueError):
    """JSON 文档不符合 v1 协议契约。"""


class ProtocolSchemaError(RuntimeError):
    """协议 schema 无法读取或编译。"""


@dataclass(frozen=True)
class DataReference:
    kind: str
    path: str
    sha256: str
    format: str


@dataclass(frozen=True)
class FactorRequest:
    schema_version: str
    request_id: str
    mode: str
    strategy_id: str
    strategy_version: str
    as_of: str
    snapshot_hash: str
    config_hash: str
    data_ref: DataReference
    params: dict[str, Any] = field(default_factory=dict)


@dataclass(frozen=True)
class RawFactors:
    ts_code: str
    momentum_60: float
    momentum_20: float
    amount_activity_20: float
    volatility_20: float
    ma20: float
    avg_amount_20_yuan: float


@dataclass(frozen=True)
class ProtocolError:
    code: str
    message: str
    ts_code: str | None = None


@dataclass(frozen=True)
class FactorResult:
    schema_version: str
    request_id: str
    strategy_id: str
    strategy_version: str
    as_of: str
    snapshot_hash: str
    status: str
    results: list[RawFactors]
    errors: list[ProtocolError]


_SCHEMA_FILES = {
    "request": "strategy-request.schema.json",
    "result": "strategy-result.schema.json",
}
_DEFAULT_SCHEMA_DIR = Path(__file__).resolve().parents[2] / "contracts"


@lru_cache(maxsize=8)
def _validator(schema_path: str) -> Draft202012Validator:
    path = Path(schema_path)
    try:
        schema = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, UnicodeDecodeError, json.JSONDecodeError) as error:
        raise ProtocolSchemaError(
            f"load protocol schema {path.name}: {error.__class__.__name__}"
        ) from error

    try:
        Draft202012Validator.check_schema(schema)
        return Draft202012Validator(schema, format_checker=FormatChecker())
    except SchemaError as error:
        raise ProtocolSchemaError(
            f"compile protocol schema {path.name}: {error.__class__.__name__}"
        ) from error


def _schema_validator(kind: str, schema_dir: Path | None) -> Draft202012Validator:
    try:
        name = _SCHEMA_FILES[kind]
    except KeyError as error:
        raise ProtocolSchemaError("unknown protocol schema") from error
    directory = schema_dir if schema_dir is not None else _DEFAULT_SCHEMA_DIR
    return _validator(str((directory / name).resolve()))


def _parse_one(payload: bytes | str) -> Any:
    def reject_constant(value: str) -> None:
        raise ValueError(f"non-standard JSON constant: {value}")

    def parse_finite_number(value: str) -> float:
        try:
            number = float(value)
        except (OverflowError, ValueError) as error:
            raise ValueError("JSON number is outside the finite float range") from error
        if not math.isfinite(number):
            raise ValueError("JSON number is outside the finite float range")
        return number

    try:
        return json.loads(
            payload,
            parse_constant=reject_constant,
            parse_float=parse_finite_number,
            parse_int=parse_finite_number,
        )
    except (json.JSONDecodeError, UnicodeDecodeError, ValueError) as error:
        raise ProtocolValidationError(
            f"decode exactly one JSON document: {error.__class__.__name__}"
        ) from error


def _validate(kind: str, payload: bytes | str, schema_dir: Path | None) -> dict[str, Any]:
    document = _parse_one(payload)
    if not isinstance(document, dict):
        raise ProtocolValidationError("protocol document must be an object")
    validator = _schema_validator(kind, schema_dir)
    errors = sorted(
        validator.iter_errors(document),
        key=lambda error: tuple(map(str, error.absolute_path)),
    )
    if errors:
        path = "/" + "/".join(map(str, errors[0].absolute_path))
        raise ProtocolValidationError(
            f"{kind} violates schema at {path}: {errors[0].validator}"
        )
    return document


def decode_request(payload: bytes | str, *, schema_dir: Path | None = None) -> FactorRequest:
    document = _validate("request", payload, schema_dir)
    return FactorRequest(
        schema_version=document["schema_version"],
        request_id=document["request_id"],
        mode=document["mode"],
        strategy_id=document["strategy_id"],
        strategy_version=document["strategy_version"],
        as_of=document["as_of"],
        snapshot_hash=document["snapshot_hash"],
        config_hash=document["config_hash"],
        data_ref=DataReference(**document["data_ref"]),
        params=document.get("params", {}),
    )


def decode_result(payload: bytes | str, *, schema_dir: Path | None = None) -> FactorResult:
    document = _validate("result", payload, schema_dir)
    return FactorResult(
        schema_version=document["schema_version"],
        request_id=document["request_id"],
        strategy_id=document["strategy_id"],
        strategy_version=document["strategy_version"],
        as_of=document["as_of"],
        snapshot_hash=document["snapshot_hash"],
        status=document["status"],
        results=[RawFactors(**item) for item in document["results"]],
        errors=[ProtocolError(**item) for item in document["errors"]],
    )


def _without_none(value: Any) -> Any:
    if isinstance(value, dict):
        return {key: _without_none(item) for key, item in value.items() if item is not None}
    if isinstance(value, list):
        return [_without_none(item) for item in value]
    return value


def _encode(kind: str, value: Any) -> bytes:
    document = _without_none(asdict(value))
    encoded = json.dumps(
        document,
        ensure_ascii=False,
        allow_nan=False,
        separators=(",", ":"),
    ).encode("utf-8")
    _validate(kind, encoded, None)
    return encoded


def encode_request(request: FactorRequest) -> bytes:
    return _encode("request", request)


def encode_result(result: FactorResult) -> bytes:
    return _encode("result", result)
