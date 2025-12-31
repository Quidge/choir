---
paths: "cmd/*.go"
---

# CLI Argument Patterns

When creating or modifying Cobra commands, follow these conventions for the `Use` field:

## Argument Documentation

| Pattern | Meaning | Example |
|---------|---------|---------|
| `CAPS` | Required positional argument | `Use: "view TASK_ID"` |
| `[brackets]` | Optional positional argument | `Use: "list [PROJECT_ID]"` |
| `[args]...` | Variadic arguments | `Use: "add [FILES]..."` |

## Examples

```go
// Required argument
var viewCmd = &cobra.Command{
    Use:   "view TASK_ID",
    Short: "View a task",
    Args:  cobra.ExactArgs(1),
}

// Optional argument
var listCmd = &cobra.Command{
    Use:   "list [PROJECT_ID]",
    Short: "List items",
    Args:  cobra.MaximumNArgs(1),
}

// No arguments
var statusCmd = &cobra.Command{
    Use:   "status",
    Short: "Show status",
}
```

## Flag Documentation

- Use short, clear descriptions
- Include "(admin only)" or similar for permission-restricted flags
- Use "(repeatable)" for slice flags
