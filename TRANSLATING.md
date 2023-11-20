# Translating rhc

rhc uses **snapcore/go-gettext** format to manage translations.

## String extraction

```shell
$ make l10n
```

This will extract strings into `.po` files and compile them into `.mo` files.
Run this after you have done changes to either source code or the translations.


## Adding a new language

```shell
$ msginit --input=po/messages.pot --output-file=po/cs.po --locale=cs_CZ.UTF-8
```


## Testing changes locally

The default locale directory is `/usr/share/locale/${LANGUAGE_CODE}/LC_MESSAGES/rhc.mo`.
To install them, run `make l10n` and copy the generated files into appropriate paths on your system:

```shell
$ sudo cp ./po/cs.mo /usr/share/locale/cs/LC_MESSAGES/rhc.mo
```
