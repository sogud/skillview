"""Exercise release installation with synthetic downloads and an isolated HOME."""
import hashlib
import io
import os
from pathlib import Path
import subprocess
import sys
import tarfile
import tempfile
import unittest


SCRIPT = Path(__file__).with_name("install-cli.sh").resolve()
VERSION = "9.8.7"
ASSET = f"skillview-cli-{VERSION}-macos-arm64.tar.gz"
BINARY = b"#!/bin/sh\necho fixture\n"


class InstallCLITests(unittest.TestCase):
    def setUp(self):
        tmp = SCRIPT.parents[1] / "tmp" / "install-cli-tests"
        tmp.mkdir(parents=True, exist_ok=True)
        self.temp = tempfile.TemporaryDirectory(dir=tmp)
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.home = self.root / "home"
        self.home.mkdir()
        self.downloads = self.root / "downloads"
        self.downloads.mkdir()
        self.bin = self.root / "tools"
        self.bin.mkdir()
        self.target = self.home / ".local" / "bin" / "skillview"
        self.calls = self.root / "curl-calls"
        self.env = dict(os.environ)
        self.env.update({
            "HOME": str(self.home),
            "TMPDIR": str(self.root),
            "PATH": str(self.bin) + os.pathsep + self.env["PATH"],
            "FIXTURE_DOWNLOADS": str(self.downloads),
            "FIXTURE_CURL_CALLS": str(self.calls),
            "FIXTURE_PLATFORM": "Darwin-arm64",
        })
        self.stub("uname", '''import os, sys
system, arch = os.environ["FIXTURE_PLATFORM"].split("-", 1)
print(system if sys.argv[1] == "-s" else arch)
''')
        self.stub("curl", '''import os, sys
from pathlib import Path
args = sys.argv[1:]
with open(os.environ["FIXTURE_CURL_CALLS"], "a") as log:
    log.write(repr(args) + "\\n")
if "--head" in args:
    print("https://github.com/sogud/skillview/releases/tag/v9.8.7", end="")
else:
    url = next(arg for arg in args if arg.startswith("https://"))
    source = Path(os.environ["FIXTURE_DOWNLOADS"]) / url.rsplit("/", 1)[1]
    if not source.is_file():
        sys.exit(22)
    output = Path(args[args.index("--output") + 1])
    output.write_bytes(source.read_bytes())
''')
        archive = self.downloads / ASSET
        with tarfile.open(archive, "w:gz") as tar:
            entry = tarfile.TarInfo("skillview")
            entry.size = len(BINARY)
            entry.mode = 0o755
            tar.addfile(entry, io.BytesIO(BINARY))
        digest = hashlib.sha256(archive.read_bytes()).hexdigest()
        self.checksum = self.downloads / (ASSET + ".sha256")
        self.checksum.write_text(f"{digest}  {ASSET}\n")

    def stub(self, name, code):
        path = self.bin / name
        path.write_text(f"#!{sys.executable}\n" + code)
        path.chmod(0o755)

    def run_installer(self, *args):
        return subprocess.run(
            ["bash", str(SCRIPT), *args], cwd=self.root, env=self.env,
            text=True, capture_output=True,
        )

    def existing(self):
        self.target.parent.mkdir(parents=True)
        self.target.write_bytes(b"keep")

    def test_installs_latest_binary_without_app(self):
        result = self.run_installer()
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(self.target.read_bytes(), BINARY)
        self.assertFalse(self.target.is_symlink())
        self.assertEqual(self.target.stat().st_mode & 0o777, 0o755)
        self.assertIn(f"/download/v{VERSION}/{ASSET}", self.calls.read_text())
        self.assertFalse((self.home / "Applications").exists())

    def test_pinned_version_and_custom_directory(self):
        target_dir = self.home / "custom-bin"
        result = self.run_installer("--version", VERSION, "--bin-dir", str(target_dir))
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual((target_dir / "skillview").read_bytes(), BINARY)
        self.assertNotIn("--head", self.calls.read_text())

    def test_dry_run_neither_downloads_nor_writes(self):
        result = self.run_installer("--dry-run")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertFalse(self.calls.exists())
        self.assertFalse(self.target.parent.exists())

    def test_existing_command_is_preserved(self):
        self.existing()
        result = self.run_installer()
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(self.target.read_bytes(), b"keep")
        self.assertFalse(self.calls.exists())

    def test_force_replaces_link_not_its_target(self):
        original = self.home / "original"
        original.write_bytes(b"keep")
        self.target.parent.mkdir(parents=True)
        self.target.symlink_to(original)
        result = self.run_installer("--force")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertFalse(self.target.is_symlink())
        self.assertEqual(self.target.read_bytes(), BINARY)
        self.assertEqual(original.read_bytes(), b"keep")

    def test_bad_checksum_preserves_existing_command(self):
        self.existing()
        self.checksum.write_text(f"{'0' * 64}  {ASSET}\n")
        result = self.run_installer("--force")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("checksum mismatch", result.stderr)
        self.assertEqual(self.target.read_bytes(), b"keep")

    def test_empty_checksum_is_rejected(self):
        self.checksum.write_text("")
        result = self.run_installer()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("Invalid release checksum", result.stderr)
        self.assertFalse(self.target.exists())

    def test_missing_asset_preserves_existing_command(self):
        self.existing()
        (self.downloads / ASSET).unlink()
        result = self.run_installer("--force")
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(self.target.read_bytes(), b"keep")

    def test_unsupported_platform_is_rejected_before_download(self):
        self.env["FIXTURE_PLATFORM"] = "Linux-x86_64"
        result = self.run_installer()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("macOS Apple Silicon", result.stderr)
        self.assertFalse(self.calls.exists())

    def test_invalid_arguments_are_rejected(self):
        for args in [("--version",), ("--version", "../../bad"), ("--unknown",)]:
            with self.subTest(args=args):
                result = self.run_installer(*args)
                self.assertEqual(result.returncode, 2)
                self.assertFalse(self.calls.exists())


if __name__ == "__main__":
    unittest.main()
