# String literals mistaken for suppression comments

The input has a valid string constant containing `gstyle-allow` before `Plain`. Suppression metadata must come from attached comment groups, not string contents.

The saved line reduction is 3 lines / 111 bytes from 17 lines / 393 bytes in 47 checks. The historical reducer checker is retired; the exact Go inputs remain here.

Run the maintained Go regressions:

```sh
go test ./adapters/goquality -run '^(TestReducedSuppressionStringIsNotAComment|TestSuppressionReadsParsedDeclarationComments)$' -count=1
```

The tests cover the reduced input, multiline strings, actual directives, malformed directives and unrelated comments.
