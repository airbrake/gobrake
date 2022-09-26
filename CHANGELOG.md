# Gobrake Changelog

## master

## [v5.6.0][v5.6.0] (July 22, 2022)

* Dependency updates

* Add custom method to set the depth of notices in apex/log and zerolog integration ([#297](https://github.com/airbrake/gobrake/pull/297))

* We have deprecated negroni middleware. It won't be maintained going forward ([#298](https://github.com/airbrake/gobrake/pull/298))

* Add in zap integration ([#299](https://github.com/airbrake/gobrake/pull/299))

* Updated minimum supported go version to 1.17 ([#300](https://github.com/airbrake/gobrake/pull/300))

* Added the feature to retry sending Error and APM data in case of failures from Airbrake Server ([#314](https://github.com/airbrake/gobrake/pull/314))

## [v5.5.2][v5.5.2] (July 22, 2022)

* dependency updates ([#275](https://github.com/airbrake/gobrake/pull/275)),
([#277](https://github.com/airbrake/gobrake/pull/277)),
([#278](https://github.com/airbrake/gobrake/pull/278)),
([#279](https://github.com/airbrake/gobrake/pull/279)),
([#280](https://github.com/airbrake/gobrake/pull/280)),
([#282](https://github.com/airbrake/gobrake/pull/282)),
([#283](https://github.com/airbrake/gobrake/pull/283)),
([#284](https://github.com/airbrake/gobrake/pull/284)),
([#285](https://github.com/airbrake/gobrake/pull/285)),
([#286](https://github.com/airbrake/gobrake/pull/286)),
([#287](https://github.com/airbrake/gobrake/pull/287)),
([#289](https://github.com/airbrake/gobrake/pull/289))

* Update Zerolog integration for automatic linking of errors to routes ([#276](https://github.com/airbrake/gobrake/pull/276))

* Update apex/log handler for automatic linking of errors to routes ([#288](https://github.com/airbrake/gobrake/pull/288))

## [v5.5.1][v5.5.1] (May 9, 2022)

* build(deps): bump github.com/valyala/fasthttp from 1.35.0 to 1.36.0 ([#267](https://github.com/airbrake/gobrake/pull/267))

* build(deps): bump github.com/jonboulle/clockwork from 0.2.3 to 0.3.0 ([#266](https://github.com/airbrake/gobrake/pull/266))

* Upgrade github.com/onsi/ginkgo to v2 ([#271](https://github.com/airbrake/gobrake/pull/271))

* build(deps): bump github.com/gofiber/fiber/v2 from 2.31.0 to 2.33.0 ([#273](https://github.com/airbrake/gobrake/pull/273))

* build(deps): bump github.com/gobuffalo/buffalo from 0.18.5 to 0.18.7 ([#272](https://github.com/airbrake/gobrake/pull/272))

* Updated changelog, notifier version ([#274](https://github.com/airbrake/gobrake/pull/274))

## [v5.5.0][v5.5.0] (May 4, 2022)

* Updated changelog, notifier version and readme ([#262](https://github.com/airbrake/gobrake/pull/262))

* Beego Status fix ([#264](https://github.com/airbrake/gobrake/pull/264))

* Fix readme link ([#268](https://github.com/airbrake/gobrake/pull/268))

* refactor status and label select statement ([#269](https://github.com/airbrake/gobrake/pull/269))

* add in zerolog integration ([#270](https://github.com/airbrake/gobrake/pull/270))

## [v5.4.0][v5.4.0] (April 13, 2022)

* Added the [Echo](https://github.com/labstack/echo) integration ([#239](https://github.com/airbrake/gobrake/pull/239))

* Added the [Iris](https://github.com/kataras/iris) integration ([#241](https://github.com/airbrake/gobrake/pull/241))

* Added the [Beego](https://github.com/beego/beego) integration ([#245](https://github.com/airbrake/gobrake/pull/245))

* Added the [net/http](https://pkg.go.dev/net/http) integration ([#246](https://github.com/airbrake/gobrake/pull/246))

* Added the [gorilla/mux](https://github.com/gorilla/mux) integration ([#249](https://github.com/airbrake/gobrake/pull/249))

* Added the [fasthttp](https://github.com/valyala/fasthttp) integration ([#252](https://github.com/airbrake/gobrake/pull/252))

* Added the [Buffalo](https://github.com/gobuffalo/buffalo) integration ([#256](https://github.com/airbrake/gobrake/pull/255))

## [v5.3.0][v5.3.0] (February 7, 2022)

* Added the [Fiber](https://github.com/gofiber/fiber) integration ([#227](https://github.com/airbrake/gobrake/pull/227)),
([#232](https://github.com/airbrake/gobrake/pull/232))

## [v5.2.0][v5.2.0] (December 13, 2021)

* Deprecated `NewMiddleware` func and replaced with `New` func in gin middleware ([#224](https://github.com/airbrake/gobrake/pull/224))

* Used [Apex/log](https://github.com/apex/log) severity levels instead of custom defined severity levels ([#225](https://github.com/airbrake/gobrake/pull/225))

## [v5.1.1][v5.1.1] (December 1, 2021)

* Updated notifier version

## [v5.1.0][v5.1.0] (December 1, 2021)

* Added [Apex/log](https://github.com/apex/log) integration ([#220](https://github.com/airbrake/gobrake/pull/220))
* Bumped gin-gonic version ([#219](https://github.com/airbrake/gobrake/pull/219))

## [v5.0.4][v5.0.4] (November 18, 2021)

* notifier: add the DisableRemoteConfig option ([#194](https://github.com/airbrake/gobrake/pull/194))
* Default Host is api.airbrake.io ([#195](https://github.com/airbrake/gobrake/pull/195))
* Updated go version and merged mod files ([#215](https://github.com/airbrake/gobrake/pull/215))

## [v5.0.3][v5.0.3] (November 17, 2020)

* Deleted support for dumping/loading the remote config
  ([#186](https://github.com/airbrake/gobrake/pull/186))
* Remote config: changed default host to `https://notifier-configs.airbrake.io`
  ([#191](https://github.com/airbrake/gobrake/pull/191))

## [v5.0.2][v5.0.2] (September 8, 2020)

* Remote config: improved error message when config cannot be requested from S3
  ([#178](https://github.com/airbrake/gobrake/pull/178))

## [v5.0.1][v5.0.1] (September 1, 2020)

* Fixed bug where `gobrake: span="http.client" is already finished gets printed`
  gets printed when a `New*Metric` method gets passed a `context.Context` which
  is also being used in multiple parallel HTTP requests
  ([#178](https://github.com/airbrake/gobrake/pull/178))
* Fixed bug where remote config doesn't respect the configured HTTP client
  ([#179](https://github.com/airbrake/gobrake/pull/179))
* Implemented a fallback mechanism for the case when remote config cannot be
  dumped loaded from the standard location. The fallback path is
  `/tmp/gobrake_remote_config.json`
  ([#180](https://github.com/airbrake/gobrake/pull/180))

## [v5.0.0][v5.0.0] (August 25, 2020)

Breaking changes:

* Deleted deprecated `KeysBlacklist` option
  ([#174](https://github.com/airbrake/gobrake/pull/174))

Bug fixes:

* None

Features:

* Added the `DisableErrorNotifications` option, which turns on/off notifications
  sent via `airbrake.Notify()` calls
  ([#147](https://github.com/airbrake/gobrake/pull/147))
* Added the `DisableAPM` option, which turns on/off notifications
  sent via `airbrake.Routes.Notify()`, `airbrake.Queues.Notify()`,
  `airbrake.Queries.Notify()` calls
  ([#148](https://github.com/airbrake/gobrake/pull/148))
* Added the `APMHost` option that sets the host to which APM data should be sent
  to ([#150](https://github.com/airbrake/gobrake/pull/150))
* Added support for remote configuration
* Added support `go 1.15` ([#168](https://github.com/airbrake/gobrake/pull/168))

## [v4.2.0][v4.2.0] (July 24, 2020)

* Added support for APM for [Negroni](https://github.com/urfave/negroni)
  ([#143](https://github.com/airbrake/gobrake/pull/143))

## [v4.1.2][v4.1.2] (July 20, 2020)

* Deprecated the `KeysBlacklist` option in favor of `KeysBlocklist`
  ([#141](https://github.com/airbrake/gobrake/pull/141))

## [v4.1.1][v4.1.1] (May 8, 2020)

* Bumped go-tdigest dependency to v3.1.0
  ([#138](https://github.com/airbrake/gobrake/pull/138))
* Bumped pkg/errors dependency to v0.9.1
  ([#138](https://github.com/airbrake/gobrake/pull/138))

## [v4.1.0][v4.1.0] (May 6, 2020)

* README was rewritten from scratch, added new information and examples
  ([#130](https://github.com/airbrake/gobrake/pull/130))
* Changed license from BSD to MIT
  ([#129](https://github.com/airbrake/gobrake/pull/129))
* Added `DisableCodeHunks` option
  ([#122](https://github.com/airbrake/gobrake/pull/122))
* Added support for go1.13 and go1.14 (started testing against them)
  ([#135](https://github.com/airbrake/gobrake/pull/135),
  [#125](https://github.com/airbrake/gobrake/pull/125))
* Improved error handling when the Airbrake API returns HTTP 400
  ([#128](https://github.com/airbrake/gobrake/pull/128))
* Started logging configuration errors
  ([#133](https://github.com/airbrake/gobrake/pull/133))

[v4.1.0]: https://github.com/airbrake/gobrake/releases/tag/v4.1.0
[v4.1.1]: https://github.com/airbrake/gobrake/releases/tag/v4.1.1
[v4.1.2]: https://github.com/airbrake/gobrake/releases/tag/v4.1.2
[v4.2.0]: https://github.com/airbrake/gobrake/releases/tag/v4.2.0
[v5.0.0]: https://github.com/airbrake/gobrake/releases/tag/v5.0.0
[v5.0.1]: https://github.com/airbrake/gobrake/releases/tag/v5.0.1
[v5.0.2]: https://github.com/airbrake/gobrake/releases/tag/v5.0.2
[v5.0.3]: https://github.com/airbrake/gobrake/releases/tag/v5.0.3
[v5.0.4]: https://github.com/airbrake/gobrake/releases/tag/v5.0.4
[v5.1.0]: https://github.com/airbrake/gobrake/releases/tag/v5.1.0
[v5.1.1]: https://github.com/airbrake/gobrake/releases/tag/v5.1.1
[v5.2.0]: https://github.com/airbrake/gobrake/releases/tag/v5.2.0
[v5.3.0]: https://github.com/airbrake/gobrake/releases/tag/v5.3.0
[v5.4.0]: https://github.com/airbrake/gobrake/releases/tag/v5.4.0
[v5.5.0]: https://github.com/airbrake/gobrake/releases/tag/v5.5.0
[v5.5.1]: https://github.com/airbrake/gobrake/releases/tag/v5.5.1
[v5.5.2]: https://github.com/airbrake/gobrake/releases/tag/v5.5.2
[v5.6.0]: https://github.com/airbrake/gobrake/releases/tag/v5.6.0
