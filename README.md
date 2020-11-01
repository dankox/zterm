# zTerm

zTerm is terminal app to manage applications run on your local PC or server with Mainframe builtin support.
Communication protocol used to connect to the server is SSH.    

- [Description](#description)
- [Configuration file](#configuration-file)
- [Theme colors](#theme-colors)
- [Keybindings](#keybindings)
- [Console commands](#console-commands)

## Description

zTerm is full terminal application where you can run multiple commands at once either from your local PC or from the remote server.    
These commands are run in seperated containers called views. Each view has a refresh interval which is used to re-run the command specified for it.    

For example, user can have view where `ps -a` would run and another view where `remote tail localhost.log` would run.    
By default refresh interval is 5s, so every five seconds `ps -a` would be executed on local PC and output displayed in the first view, 
and `tail localhost.log` would be executed on remote server and its output would be displayed in the second view.

This allows to execute multiple different commands where output is changed in time and monitor them (as in the example with log file).

Views are stacked on each other from top to bottom. Top view has the smallest position number (e.g.: 1), while bottom view has the largest position number.
Position number can be changed in configuration file.

zTerm has builtin console, where user can configure what should be displayed in views and highlight specific words or lines.    
Commands can be executed in console without attaching them to specific view. In such case the output would be visible in the console. 
Autocompletion with `Tab` key is provided with some basic functionality for internal commands.


## Configuration file

Sample of configuration file which can be used to setup your zterm before first run.

```yaml
server:
  host: localhost
  user: userid
theme:
  color-space: basic
  console: 6
  error: red
  frame: 2
  frame-select: 3
  highlight: magenta
views:
  joblog:
    position: 1
    size: 20
    job: remote zjobs
    hiline:
    - "ERROR"
    hiword:
    - userid
  syslog:
    position: 2
    size: 70
    job: remote zsyslog
    hiline:
    - userid
    hiword: []
  mytop:
    position: 3
    size: 10
    job: ps -a
```

Configuration can be created also by running `savecfg` in the `zterm` console. However, theme colors are not supported yet (need to be setup in config file).     
Here is an example how to do it from zTerm.

```bash
zterm --user userid host

# hit ` or Esc to open application console

# following commands are executed in the application console
addview joblog
addview syslog
addview mytop
resize joblog +10
resize syslog +70
attach joblog remote zjobs
view joblog hi-line ERROR
view joblog hi-word userid
attach syslog remote zsyslog
view joblog hi-line userid
attach mytop ps -a
savecfg
```

## Theme colors

Colors in zTerm can be setup in configuration file. 

```yaml
theme:
# color-space can have values basic, ansi256, truecolor
  color-space: basic
# all following keywords have color as a value
  console: 6
  error: red
  frame: 2
  frame-select: 3
  highlight: magenta
```

There are 3 color spaces available
1. basic - allows colors from 0 - 15 from ANSI256 colors which are translated into 30-37 (foreground ansi) and the high-intensity colors (8-15) get `faint` parameter (`\x1b[2m`)
2. ansi256 - allows colors from ANSI256 space, which is 0 - 255 (where 0-15 basic colors, 16 - 231 ansi colors, 232 - 255 grayscale colors)
3. truecolor - allows 24bit colors (if your terminal supports it). 

All the colors can be specified either by ANSI color code (0-255) or by hex values ("#RRGGBB") or by name recognizable by [colornames](https://godoc.org/golang.org/x/image/colornames) golang package.     
Colors can be specified by hex values or color names even for `basic` or `ansi256` color space. Theme namanger will try to convert them in best possible way to correspond to the color allowed in specified color space. The same applies other way around (from `basic` to `truecolor`).

## Keybindings

Keybind | Description
---|---
`F1` | Help popup - displays basic help
`F10` or `Ctrl+C` | Quit application
"\`" | Open console 
`Esc` | Open console and close console, close pop-up window (like help)
`Tab` | Cycle thru views, select next one. It does work only on views in stack, not on console or pop-up
`Tab` in console | Autocompletion function. It allows simple autocompletion to commands (just basic stuff)
`Ctrl+R` | Change refresh rate on selected view. It cycle thru 2s, 5s and 10s refresh rate.
`Ctrl+Z` | Stop refreshing selected view.

## Console commands

Command | Description
--- | ---
`addview` | Add a new view to the bottom of the view stack. If no view was added before first view will be inserted.<br>Usage: `addview <view-name>`
`attach` | Attach a command to the specified view. It can be regular command or `remote` command. <br>Usage: `attach <view-name> <command>`
`exit` | Exit zTerm. No mather what is running, everything will be stop and application will be closed.
`help` | Display available commands.
`remote` | Run command on server (if connected to server).<br>Usage: `remote <command>`
`resize` | Resize view by specific number. Negative number shrink view, while positive enlarge view.<br>Usage: `resize <view-name> <number>`
`savecfg` | Save current zTerm application setup into configuration file. It saves view setup and connection setup.
`view` | Configure specified view. Current configuration commands are `hi-line`, `hi-word` for highlighting output in the view and `hi-remove` for removing highlight.<br>Usage: `view <view-name> [hi-line\|hi-word\|hi-remove] [arg]`
