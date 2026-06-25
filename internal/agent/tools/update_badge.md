Update the session badge — a persistent context note that is automatically re-injected into the conversation periodically and after context compaction.

Use the badge to record:
- Key task goals or constraints for the current session
- Important decisions or architecture choices that must not be forgotten
- Critical information that should survive context compaction

You MUST call `read_badge` at least once before calling this tool. The badge replaces any previous content entirely.

Keep the badge concise (ideally under 500 characters). If the badge is no longer needed, set it to an empty string to clear it.
