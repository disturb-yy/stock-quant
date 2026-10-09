import unittest
from pathlib import Path
from tempfile import TemporaryDirectory
from unittest.mock import patch

from check_secrets import find_secrets, format_finding, scan_root


class CheckSecretsTests(unittest.TestCase):
    def test_empty_and_placeholder_values_are_ignored(self):
        content = """TUSHARE_TOKEN=
DB_PASSWORD=${DB_PASSWORD}
token: <SECRET>
"""
        self.assertEqual(find_secrets(content), [])

    def test_assigned_secret_is_detected_without_returning_value(self):
        secret = "X" * 40
        content = "TUSHARE_TOKEN=" + secret

        findings = find_secrets(content)
        self.assertEqual(findings, [(1, "TUSHARE_TOKEN")])
        message = format_finding(".env", *findings[0])
        self.assertIn(".env:1", message)
        self.assertIn("TUSHARE_TOKEN", message)
        self.assertNotIn(secret, message)

    def test_database_dsn_is_detected(self):
        key = "MYSQL" + "_DSN"
        value = "X" * 40
        content = key + "=" + value
        self.assertEqual(find_secrets(content), [(1, "MYSQL_DSN")])

    def test_missing_scan_root_is_not_reported_as_clean(self):
        with self.assertRaises(NotADirectoryError):
            scan_root(Path("/definitely-not-a-stock-quant-directory"))

    def test_unreadable_file_blocks_the_scan(self):
        with TemporaryDirectory() as directory:
            root = Path(directory)
            (root / "config.env").write_text("", encoding="utf-8")
            with patch.object(Path, "read_text", side_effect=PermissionError("denied")):
                with self.assertRaises(PermissionError):
                    scan_root(root)


if __name__ == "__main__":
    unittest.main()
