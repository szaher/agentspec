"""Setup for the IntentLang Pygments lexer plugin."""

from setuptools import setup

setup(
    name="pygments-intentlang",
    version="1.0.0",
    description="Pygments lexer for IntentLang (.ias) files",
    packages=["pygments_intentlang"],
    package_dir={"pygments_intentlang": "."},
    entry_points={
        "pygments.lexers": [
            "intentlang = pygments_intentlang.lexer:IntentLangLexer",
        ],
    },
    install_requires=["Pygments>=2.16.0"],
)
