# Changelog

## [1.7.1](https://github.com/grafana/pyroscope-lambda-extension/compare/v1.7.0...v1.7.1) (2025-01-30)


### Bug Fixes

* deprecate release-please manual workflow ([#37](https://github.com/grafana/pyroscope-lambda-extension/issues/37)) ([e5c72d6](https://github.com/grafana/pyroscope-lambda-extension/commit/e5c72d61c1a6259f0cb36e27db3df01db5116f6f))

## [1.7.0](https://github.com/grafana/pyroscope-lambda-extension/compare/v1.6.1...v1.7.0) (2023-12-08)


### Features

* inject session ID in the client ([#32](https://github.com/grafana/pyroscope-lambda-extension/issues/32)) ([56ca696](https://github.com/grafana/pyroscope-lambda-extension/commit/56ca69668b90ee2fb2c6a1c81ecd88298776426c))

## [1.6.1](https://github.com/grafana/pyroscope-lambda-extension/compare/v1.6.0...v1.6.1) (2023-10-25)


### Bug Fixes

* relaying error handling ([#30](https://github.com/grafana/pyroscope-lambda-extension/issues/30)) ([815cae0](https://github.com/grafana/pyroscope-lambda-extension/commit/815cae0407880d3d3143584f73e31c58aaba3d98))

## [1.6.0](https://github.com/grafana/pyroscope-lambda-extension/compare/v1.5.0...v1.6.0) (2023-10-06)


### Features

* update pyroscope-go ([976cc87](https://github.com/grafana/pyroscope-lambda-extension/commit/976cc87e984ee173203ab3b8e7f89d8e8fccc206))

## [1.5.0](https://github.com/grafana/pyroscope-lambda-extension/compare/v1.4.1...v1.5.0) (2023-08-11)


### Features

* add optional json logging format for advanced configurations ([#25](https://github.com/grafana/pyroscope-lambda-extension/issues/25)) ([2389785](https://github.com/grafana/pyroscope-lambda-extension/commit/238978539791f2c61f6c9ac86cad372614a6bc0f))


### Bug Fixes

* change log level to debug when exiting ([#26](https://github.com/grafana/pyroscope-lambda-extension/issues/26)) ([c3bf02f](https://github.com/grafana/pyroscope-lambda-extension/commit/c3bf02f062f2eabe309f6e8e8aa01eb2192fd566))

## [1.4.1](https://github.com/grafana/pyroscope-lambda-extension/compare/v1.4.0...v1.4.1) (2023-07-13)


### Bug Fixes

* rename org_id to tenant_id ([#23](https://github.com/grafana/pyroscope-lambda-extension/issues/23)) ([f21acfc](https://github.com/grafana/pyroscope-lambda-extension/commit/f21acfccdcb1dfac1b9234473b1fc47e87a72f79))

## [1.4.0](https://github.com/pyroscope-io/pyroscope-lambda-extension/compare/v1.3.0...v1.4.0) (2023-04-23)


### Features

* relay to phlare ([#21](https://github.com/pyroscope-io/pyroscope-lambda-extension/issues/21)) ([bb9ac34](https://github.com/pyroscope-io/pyroscope-lambda-extension/commit/bb9ac34129d5da5eb0913a4e594dd08de4995087))

## [1.3.0](https://github.com/pyroscope-io/pyroscope-lambda-extension/compare/v1.2.0...v1.3.0) (2022-10-12)


### Features

* flush queue before next event polling ([#18](https://github.com/pyroscope-io/pyroscope-lambda-extension/issues/18)) ([e5b06d2](https://github.com/pyroscope-io/pyroscope-lambda-extension/commit/e5b06d2e38d174daa52828e45fb7783700bd86ee))


### Bug Fixes

* configure MaxIdleConnsPerHost (was 2 by default) ([#19](https://github.com/pyroscope-io/pyroscope-lambda-extension/issues/19)) ([9214f26](https://github.com/pyroscope-io/pyroscope-lambda-extension/commit/9214f26e2e1b4ae460981ab1f09f01c6ac92f201))

## [1.2.0](https://github.com/pyroscope-io/pyroscope-lambda-extension/compare/v1.1.0...v1.2.0) (2022-10-04)


### Features

* allow NUM_WORKERS to be configurable ([#12](https://github.com/pyroscope-io/pyroscope-lambda-extension/issues/12)) ([efbe468](https://github.com/pyroscope-io/pyroscope-lambda-extension/commit/efbe4680175be80f4db2d0ce0e3b301443d8201e))
* allow timeout to be configurable ([#16](https://github.com/pyroscope-io/pyroscope-lambda-extension/issues/16)) ([8f80d90](https://github.com/pyroscope-io/pyroscope-lambda-extension/commit/8f80d9071e352362df82dbecabb8b086494beaac))

## [1.1.0](https://github.com/pyroscope-io/pyroscope-lambda-extension/compare/v1.0.1...v1.1.0) (2022-07-05)


### Features

* initial version ([#1](https://github.com/pyroscope-io/pyroscope-lambda-extension/issues/1)) ([31471b4](https://github.com/pyroscope-io/pyroscope-lambda-extension/commit/31471b4fd059f511720baf6dba2e04a7236083ca))


### Bug Fixes

* remove ap-northeast{2,3} arm builds ([#8](https://github.com/pyroscope-io/pyroscope-lambda-extension/issues/8)) ([88501e9](https://github.com/pyroscope-io/pyroscope-lambda-extension/commit/88501e9ea03ab9fc1dcdf673ba341279679afc73))
* set architecture when publishing ([#6](https://github.com/pyroscope-io/pyroscope-lambda-extension/issues/6)) ([ae3372f](https://github.com/pyroscope-io/pyroscope-lambda-extension/commit/ae3372f4697a1d97246c0eb73448bf752ec370a1))

## 1.0.0 (2022-07-04)


### Features

* initial version ([#1](https://github.com/pyroscope-io/pyroscope-lambda-extension/issues/1)) ([31471b4](https://github.com/pyroscope-io/pyroscope-lambda-extension/commit/31471b4fd059f511720baf6dba2e04a7236083ca))
