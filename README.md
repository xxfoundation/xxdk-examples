# xxdk-examples

This repository contains examples in different programming languages
for working with xx network's `xxdk`. Libraries are available for
the following systems and languages:

1. Desktop: Golang (full), C# (partial)
2. Web (Chrome/Safari): Javascript (Client side rendering only),
   Typescript (Client side rendering only)
3. Android: Kotlin, Java
4. iOS: Swift, Objective C

Libraries for other languages are planned. At this time, we are
looking at completing C# support, Javascript server side in Node, and
Rust. We are also in the process of reworking the design to simplify
deployment in different languages.

## Organization

Each folder contains an example project in the provided language:

1. `android` - Kotlin example written with Android Studio
2. `iOS` - Swift example written with XCode
3. `reactjs` - Javascript example written with Visual Studio Code and
   React framework.
4. `golang` - Golang example for the command line. This is useful for
   writing bots.

Please see the README inside each folder for platform specific
details. The examples all use the Direct Messaging module, which is
specified at the elixxir/docs repository:

https://git.xx.network/elixxir/docs/-/blob/1ce87f00db92fba7b0fb09d4a4cf22d0f7815ac2/dm.md

There are several modules developers can use. The `xxdk` is complex
software. The primary repositories to explore it are:

1. https://git.xx.network/elixxir/client - Android/iOS/Golang. The
   primary client development library.
2. https://git.xx.network/elixxir/xxdk-wasm - Javascript library and
   npm packaging.
3. https://git.xx.network/elixxir/libxxdk - C library bindings and C#
   implementation.

While the library is somewhat mature, the API itself is not stable and
is subject to change. We will post transition guides to the README and
this repository when that happens.

## Support

Please file issues on the project at `git.xx.network`.

There is a forum for questions here:

https://forum.xx.network/

You can find us on Discord here:

https://discord.com/invite/kbS4dSrPCv

## Contributing

We are open to contributions, please open a Merge Request / Pull Request
and we will take a look.

## Authors and acknowledgment

We appreciate everyone who has contributed to this project. The key
individuals are:

- Richard Carback, main author
- Sidhant Sharma, feedback
- Matthieu Bertin, feedback

## License

These examples are licensed under 2-Clause BSD. See `LICENSE.md` for
more information.

## Project status

These examples were last updated for the xxdk client library
for Golang, iOS, and Android for this version:

```
v4.7.2
```

The xxdk-wasm package used was:

```
v0.3.19
```

The NuGet package version used was:

```
# TODO: We have one but it needs to be added.
```
