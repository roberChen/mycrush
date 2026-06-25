Read the current session badge. The badge is persistent context that the model should keep in mind throughout the session — important constraints, task goals, key decisions, or critical reminders. The badge is automatically re-injected into the conversation periodically and after context compaction.

You MUST call this tool before calling `update_badge` to ensure you are aware of the current badge content before modifying it.
