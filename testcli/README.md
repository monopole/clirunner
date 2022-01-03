# testcli

testcli is an interactive command line interpreter (CLI) solely intended for use
in testing the nearby CLI ProcRunner.

It reads commands from `stdin` and prints stuff to stdout and has no other side
effects. It pretends to wrap a database and perform database queries, like
mysql.

To see flags and commands:

```
testcli help
```

The flags can be used to change the CLI's behavior, e.g. cause it to error or
take a long time to respond to commands.
