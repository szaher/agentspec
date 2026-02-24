"""Setup for the IntentLang Pygments lexer plugin."""

from setuptools import setup, find_packages

setup(
    name="pygments-intentlang",
    version="1.0.0",
    description="Pygments lexer for IntentLang (.ias) files",
    packages=find_packages(),
    entry_points={
        "pygments.lexers": [
            "intentlang = pygments_intentlang.lexer:IntentLangLexer",
        ],
    },
    install_requires=["Pygments>=2.16.0"],
)
