## cr completion fish

Generate the autocompletion script for fish

### Synopsis

Generate the autocompletion script for the fish shell.

To load completions in your current shell session:

	cr completion fish | source

To load completions for every new session, execute once:

	cr completion fish > ~/.config/fish/completions/cr.fish

You will need to start a new shell for this setup to take effect.


```
cr completion fish [flags]
```

### Options

```
  -h, --help              help for fish
      --no-descriptions   disable completion descriptions
```

### Options inherited from parent commands

```
      --config string   Config file (default is $HOME/.cr.yaml)
```

### SEE ALSO

* [cr completion](cr_completion.md)	 - Generate the autocompletion script for the specified shell

