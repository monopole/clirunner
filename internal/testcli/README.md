# testcli

`testcli` is an interactive command line interpreter (CLI) program that
looks like it wraps a database (like mysql).  It's intended for use
in testing the nearby CLI ProcRunner.

It reads commands from `stdin` and prints stuff to stdout and has no
other side effects.

To see flags and commands:

```
testcli help
```

The flags can be used to change the CLI's behavior, e.g. cause it
to error when  reading a particular database row, or take a long
time to do a query.
