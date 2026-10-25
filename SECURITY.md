# Security policy

## Reporting a vulnerability

Please **do not** report security vulnerabilities in public GitHub issues.

Report them privately through [GitHub Security Advisories](https://github.com/haraig/spellcast/security/advisories/new) instead. Include as much detail as you can:

- a description of the vulnerability,
- steps to reproduce it,
- the possible impact,
- a suggested fix (optional).

You can expect an acknowledgement within 48 hours and a resolution or a status update within 14 days.

## What is in scope

Spellcast runs on your desktop with your user's rights. Areas where a security problem matters most:

- **The keyboard hook:** it must not store, log or send keys. See [The keyboard hook and security](docs/explanation/keyboard-hook-and-security.md).
- **The clipboard:** Spellcast copies the selection and pastes the answer, and must restore your previous clipboard content.
- **The API key:** it is stored in the Windows Credential Manager and must never appear in a file or the log.
- **Prompt injection:** the selected text goes to the model. Text that makes Spellcast do more than answer (for example, act on other apps) is a vulnerability.

## Supported versions

Only the latest release receives security fixes.
