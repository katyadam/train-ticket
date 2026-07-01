from __future__ import annotations

import subprocess
import sys
from pathlib import Path


def test_machine_readable_architecture_oracle():
    root = Path(__file__).resolve().parents[1]
    subprocess.run([sys.executable, str(root / "benchmark-ground-truth" / "verify.py")], cwd=root, check=True)
