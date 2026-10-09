# Security

This file is the template's own: a repository started from it deletes it or writes its
own.

## Reporting a vulnerability

Write to **info@8wonders.de**. Do not open a public issue or pull request for it.

Say what you found, the version, and how to see it happen: the command or the test, with
every secret taken out, and what it did that it should not.

You get an answer within three working days. We tell you what we found, fix what is a
vulnerability in a new release, and publish an advisory that credits you unless you
would rather it did not.

## What is a vulnerability here

A flaw in the template that every integration started from it copies:

- `acme-example` writes the token, or a secret of the settings, anywhere but its answer
  on standard output: an error line among them.
- It accepts the secret `token` on its command line.
- It reads a token file that is a symbolic link, is not a regular file, belongs to
  another user, or that group or others may read.
- Its answer claims a path of another project than the argument's, or a host the
  description does not list.

## What is not

- A vulnerability in an integration started from the template: report it to that
  integration's repository. When the flaw came from the template, report it here too.
- The contract, the conformance checks and the shared workflows are
  [qoryai/integrations'](https://github.com/qoryai/integrations/blob/main/SECURITY.md).
- Forager, and what it does with a credential role's answer, are
  [Forager's](https://github.com/qoryai/forager/blob/main/SECURITY.md).

If you are not sure which side something falls on, write anyway.
