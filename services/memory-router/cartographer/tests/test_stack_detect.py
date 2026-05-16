"""Stack-detector tests."""

from __future__ import annotations

from pathlib import Path

from crucible_cartographer.stack_detect import detect


def test_detect_nextjs(tmp_path: Path) -> None:
    (tmp_path / "package.json").write_text('{"dependencies":{"next":"^14.2"}}')
    (tmp_path / "next.config.js").write_text("module.exports = {};")
    res = detect(str(tmp_path))
    assert res.primary == "nextjs"
    assert res.versions.get("next") == "14.2"


def test_detect_fastapi(tmp_path: Path) -> None:
    (tmp_path / "pyproject.toml").write_text('[project]\nname="x"\ndependencies = ["fastapi>=0.110"]\n')
    res = detect(str(tmp_path))
    assert res.primary == "fastapi"


def test_detect_django(tmp_path: Path) -> None:
    (tmp_path / "manage.py").write_text("#!/usr/bin/env python\n")
    (tmp_path / "requirements.txt").write_text("django>=5.0\n")
    res = detect(str(tmp_path))
    assert res.primary == "django"


def test_detect_go(tmp_path: Path) -> None:
    (tmp_path / "go.mod").write_text("module acme/svc\ngo 1.22\n")
    res = detect(str(tmp_path))
    assert res.primary == "go_services"


def test_detect_rust(tmp_path: Path) -> None:
    (tmp_path / "Cargo.toml").write_text('[package]\nname = "x"\nversion = "0.1.0"\n')
    res = detect(str(tmp_path))
    assert res.primary == "rust_services"


def test_detect_empty_repo(tmp_path: Path) -> None:
    res = detect(str(tmp_path))
    assert res.primary == ""
    assert res.confidence == 0.0


def test_detect_nextjs_plus_fastapi_monorepo(tmp_path: Path) -> None:
    (tmp_path / "package.json").write_text('{"dependencies":{"next":"14"}}')
    (tmp_path / "pyproject.toml").write_text('[project]\ndependencies = ["fastapi"]\n')
    res = detect(str(tmp_path))
    assert res.primary == "nextjs"
    assert "fastapi" in res.secondary
