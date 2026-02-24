"""IntentLang Pygments Lexer for syntax highlighting in documentation."""

from pygments.lexer import RegexLexer, bygroups, words
from pygments.token import (
    Comment,
    Keyword,
    Name,
    Number,
    Operator,
    Punctuation,
    String,
    Text,
    Literal,
)


class IntentLangLexer(RegexLexer):
    """Pygments lexer for IntentLang (.ias) files."""

    name = "IntentLang"
    aliases = ["intentlang", "ias"]
    filenames = ["*.ias"]

    tokens = {
        "root": [
            # Comments
            (r"#.*$", Comment.Single),
            (r"//.*$", Comment.Single),
            # Strings (double-quoted with escapes)
            (r'"(?:\\.|[^"\\])*"', String.Double),
            # Block keywords (top-level resource types)
            (
                words(
                    (
                        "package",
                        "version",
                        "lang",
                        "agent",
                        "prompt",
                        "skill",
                        "tool",
                        "deploy",
                        "pipeline",
                        "step",
                        "type",
                        "server",
                        "client",
                        "secret",
                        "environment",
                        "policy",
                        "plugin",
                    ),
                    suffix=r"\b",
                ),
                Keyword.Declaration,
            ),
            # Reference and relation keywords
            (
                words(
                    (
                        "uses",
                        "connects",
                        "exposes",
                        "delegate",
                        "to",
                        "when",
                        "from",
                        "target",
                    ),
                    suffix=r"\b",
                ),
                Keyword.Namespace,
            ),
            # Attribute keywords
            (
                words(
                    (
                        "model",
                        "strategy",
                        "max_turns",
                        "timeout",
                        "token_budget",
                        "temperature",
                        "stream",
                        "on_error",
                        "max_retries",
                        "fallback",
                        "content",
                        "variables",
                        "description",
                        "input",
                        "output",
                        "method",
                        "url",
                        "headers",
                        "body_template",
                        "binary",
                        "args",
                        "language",
                        "code",
                        "transport",
                        "command",
                        "auth",
                        "port",
                        "image",
                        "namespace",
                        "replicas",
                        "health",
                        "autoscale",
                        "resources",
                        "cpu",
                        "memory",
                        "path",
                        "interval",
                        "min_replicas",
                        "max_replicas",
                        "metric",
                        "depends_on",
                        "parallel",
                        "env",
                        "secrets",
                        "default",
                        "allow",
                        "deny",
                        "require",
                        "store",
                    ),
                    suffix=r"\b",
                ),
                Name.Attribute,
            ),
            # Type keywords
            (
                words(
                    ("string", "int", "bool", "number", "enum", "list"),
                    suffix=r"\b",
                ),
                Keyword.Type,
            ),
            # Modifier keywords
            (words(("required",), suffix=r"\b"), Keyword.Pseudo),
            # Boolean literals
            (words(("true", "false"), suffix=r"\b"), Literal),
            # Tool variant keywords
            (
                words(("mcp", "http", "inline"), suffix=r"\b"),
                Keyword.Constant,
            ),
            # Numbers (integer and float)
            (r"\b\d+\.\d+\b", Number.Float),
            (r"\b\d+\b", Number.Integer),
            # Punctuation
            (r"[{}()\[\]]", Punctuation),
            # Template variables
            (r"\{\{[a-zA-Z_][a-zA-Z0-9_]*\}\}", Name.Variable),
            # Whitespace
            (r"\s+", Text),
            # Identifiers (catch-all)
            (r"[a-zA-Z_][a-zA-Z0-9_-]*", Name),
        ],
    }
