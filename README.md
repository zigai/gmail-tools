# gmail-tools

[![tests](https://img.shields.io/github/actions/workflow/status/zigai/gmail-tools/test.yml?branch=master&label=tests&labelColor=30363d)](https://github.com/zigai/gmail-tools/actions/workflows/test.yml)
[![go](https://img.shields.io/github/go-mod/go-version/zigai/gmail-tools?label=&logo=go&logoColor=white&labelColor=30363d)](go.mod)
[![license](https://img.shields.io/github/license/zigai/gmail-tools?label=license&labelColor=30363d)](LICENSE)

Give an AI agent your Gmail and Google Drive across several Google accounts. It can search, read, label, and clean up mail, and it can't send mail, share files, or change forwarding.

- **Search:** Find mail in every account with one query, newest first.
- **Inbox cleanup:** Rank the senders and mailing lists that fill your inbox.
- **Filter audit:** See which Gmail filters still match mail and which are dead.
- **Filter drafts:** Get ready-to-paste [`gmailctl`](https://github.com/mbrt/gmailctl) rules for your noisiest senders.
- **Mail and Drive:** Run any [`gog`](https://github.com/openclaw/gogcli) Gmail or Drive command with the risky ones blocked.

## Installation

```sh
go install github.com/zigai/gmail-tools@latest
```

Requires Go 1.27 or newer, `gog`, and `gmailctl`. Sign in to each account with `gog auth add <email> --services gmail,drive`.

## Quick start

Name your accounts in `~/.config/gmail-tools/config.toml`. The `gmailctl_config` folder is optional and enables the `filters` commands:

```toml
default_account = "personal"

[accounts.personal]
email = "me@example.com"
gmailctl_config = "~/.gmailctl"

[accounts.work]
email = "me@work.example"
```

Check that every account works, then search all of them:

```sh
gmail-tools accounts
gmail-tools search "invoice newer_than:30d"
```

Find dead filters and draft rules for the senders that fill the inbox, then apply your edited config:

```sh
gmail-tools filters audit --unused
gmail-tools filters suggest >> drafts.jsonnet
gmail-tools filters diff && gmail-tools filters apply
```

## Commands

```text
$ gmail-tools --help
Search, triage, and filter Gmail across several Google accounts.

Accounts are read from /home/user/.config/gmail-tools/config.toml.
mail and drive run gog with sending, sharing, permanent deletion, and changes
to forwarding, delegation, send-as, and vacation settings blocked. Filters
change only through gmailctl, so its config stays the source of truth.

Usage:
  gmail-tools [flags]
  gmail-tools [command]

Available Commands:
  accounts    List accounts with their sign-in state and unread inbox mail
  drive       Run a gog drive command for an account
  filters     Audit Gmail filters, draft new ones, and apply them with gmailctl
  help        Help about any command
  mail        Run a gog gmail command for an account
  search      Search mail in every account, newest first
  senders     Rank the senders and mailing lists that fill the inbox

Flags:
  -a, --account string   Account name from the config file (default: default_account)
  -h, --help             help for gmail-tools
  -v, --version          Print version

Use "gmail-tools [command] --help" for more information about a command.
```

## License

[MIT](LICENSE)
