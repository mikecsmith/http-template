# Changelog

## [0.0.3](https://github.com/mikecsmith/http-template/compare/v0.0.2...v0.0.3) (2026-04-15)


### Bug Fixes

* data race in logger WithAttrs ([412cadb](https://github.com/mikecsmith/http-template/commit/412cadbdfa7637c9741d2601a653bf454649a0af))
* data race in logger WithAttrs ([0f123c1](https://github.com/mikecsmith/http-template/commit/0f123c1557a9b4321e31a1e169e8f934ae2b507d))
* independent contexts for metrics and server ([1a3f708](https://github.com/mikecsmith/http-template/commit/1a3f708cde9f8f6a78eab0a6e0a855bf0b740b8f))
* independent contexts for metrics and server ([cdf0c65](https://github.com/mikecsmith/http-template/commit/cdf0c658a7c6cbd75ac7f2e57824b7cdba18c9ed))
* request timeouts != read timeouts ([6144324](https://github.com/mikecsmith/http-template/commit/6144324663fe2f6a8e0c07475973ff04a1496c3c))
* request timeouts != read timeouts ([451158e](https://github.com/mikecsmith/http-template/commit/451158e30cb2e6c9a8b249717de002d03c825bd5))

## [0.0.2](https://github.com/mikecsmith/http-template/compare/v0.0.1...v0.0.2) (2026-04-10)


### Features

* add basic hello world http endpoint using respond.go ([3895571](https://github.com/mikecsmith/http-template/commit/3895571cc3a005aa7b866ace189d14a2c25af976))
* add contextual structured logging implementation ([a313347](https://github.com/mikecsmith/http-template/commit/a3133470fd6c79f949e2a66ee84dbf6727ec0589))
* add demo HelloWorld handlers and revise respond package ([c558134](https://github.com/mikecsmith/http-template/commit/c5581342b420fe25b9eb0e4a96cc981ff30abe50))
* add descriptions to logger_test.go suite - structured logging in Go is interesting... ([8781339](https://github.com/mikecsmith/http-template/commit/87813394e5d077e5e290c59216f0a6481f5b24d5))
* add distroless Dockerfile for goreleaser ([3c12b7e](https://github.com/mikecsmith/http-template/commit/3c12b7ed13b7e8d7b7af3851d346dfe62982c264))
* add echo endpoint to reflect GET and POST requests ([e12b317](https://github.com/mikecsmith/http-template/commit/e12b3172a1f49d40a086a8f22bcccfe38c4c854b))
* add errgroup and go sync package to run for template ([46327f2](https://github.com/mikecsmith/http-template/commit/46327f2b3631c01e7478876355c3aee98514438c))
* add goreleaser config with multi-arch docker images ([c2fcaa8](https://github.com/mikecsmith/http-template/commit/c2fcaa889574b8285f3d27b3994e1aea2ff9d39b))
* add healthz and readyz endpoints ([a1cc3e5](https://github.com/mikecsmith/http-template/commit/a1cc3e5a0c1a64e384b0d390fa843b64b6286777))
* add httplab respond.go and tests ([fa32c5e](https://github.com/mikecsmith/http-template/commit/fa32c5e583552e7f304aae798aa71328c5ed6211))
* add log level config ([77de464](https://github.com/mikecsmith/http-template/commit/77de464d95436b3011eb07ae9e009923266bc505))
* add logger_test claude generated tests with koans to aid learning ([6406da6](https://github.com/mikecsmith/http-template/commit/6406da6055103969f6ef4c73cad1cf3033c431ff))
* add logging middleware ([6fa9f28](https://github.com/mikecsmith/http-template/commit/6fa9f28e13a2f5629b95b0028e31d214d58c07e7))
* add metrics plumbing with otel http instrumentation ([f26a7c9](https://github.com/mikecsmith/http-template/commit/f26a7c9c2137f3670653a3ee5fa90db7df2478e5))
* add middleware skeleton for hydrating logger context and utils for chaining middleware ([b3fc44e](https://github.com/mikecsmith/http-template/commit/b3fc44ef537a2574b97be212219b48e3200c6ffd))
* add parseConfig and NotFound route ([4f081d0](https://github.com/mikecsmith/http-template/commit/4f081d0e8db3746cfc63304a08b6acf03c142d98))
* Add README's to projects ([6f81b4f](https://github.com/mikecsmith/http-template/commit/6f81b4f77654899f6282a59d8e3fdbba516e29e4))
* add request decoding package ([eb673c7](https://github.com/mikecsmith/http-template/commit/eb673c728f88765e8eac943ae807e55d52ea1127))
* add secure headers middleware ([d923f05](https://github.com/mikecsmith/http-template/commit/d923f050e5f8b2be33455ebc80e9bd5e0092ad55))
* add Tiltfile and ctlptl kind cluster for local dev ([12b1ddb](https://github.com/mikecsmith/http-template/commit/12b1ddbadf87105c084f76d8e2b1162322c4b788))
* add timeout values to config ([7e68455](https://github.com/mikecsmith/http-template/commit/7e68455118025b753d0eb29820286a75f90a1c13))
* always print server startup banner regardless of log level ([5f2e812](https://github.com/mikecsmith/http-template/commit/5f2e812825e83b4c82750e8bd60c5af7aeea2325))
* collapse middleware chain and wire request logging ([e04a7d9](https://github.com/mikecsmith/http-template/commit/e04a7d9bd3a558c41c17de0cbaf4565c9ef0a751))
* derive otel service name from binary ([93844d4](https://github.com/mikecsmith/http-template/commit/93844d46bfc3fc77c4f515f22da16e57b96e5000))
* dynamic tiltfile and mise setup ([0d72d08](https://github.com/mikecsmith/http-template/commit/0d72d08ec12e1776506873be2b2f2796c65573b6))
* enable writer injection in logger.Init() and thread through to run ([43679f6](https://github.com/mikecsmith/http-template/commit/43679f69c474a3a0cec0280471cde52e60a1f979))
* expose build version commit and date in startup log ([3bd6c72](https://github.com/mikecsmith/http-template/commit/3bd6c728fdb26fdd835e903948d75fa009aa8b10))
* front the dev cluster with Traefik Gateway API and mkcert TLS ([4b0650b](https://github.com/mikecsmith/http-template/commit/4b0650b8967ec0eb7c9cf8aebbdf1bdd715b2104))
* implement contextual logging middleware and tests ([ac327ea](https://github.com/mikecsmith/http-template/commit/ac327ea7cefad37e89f3b4909fbb55e8085a6186))
* implement logger_test.go solutions ([2db2054](https://github.com/mikecsmith/http-template/commit/2db2054145b6de58592f3d8406820a60bf9db646))
* implement secure header middleware ([f8e0590](https://github.com/mikecsmith/http-template/commit/f8e0590f54b3596e8b7f8f00bae6bbfb0315e94b))
* migrate goreleaser to dockers_v2 ([164e05c](https://github.com/mikecsmith/http-template/commit/164e05c7df957cd126abc316f8cee5ac882b3b3f))
* move probe endpoints to debug level logging ([f2223eb](https://github.com/mikecsmith/http-template/commit/f2223eb487951d41d3a79f39c2386b3320c20fd4))
* update middleware to inject and capture request ids ([740a7e1](https://github.com/mikecsmith/http-template/commit/740a7e1dd0d03897cfcb933bec19faa1c34091bd))
* use NewFlagSet and thread args from main into run ([1171337](https://github.com/mikecsmith/http-template/commit/1171337ce3d20dbd453b6cf11b1bd9d7dcdc60a1))


### Bug Fixes

* broaden release please triggers ([eaa6d06](https://github.com/mikecsmith/http-template/commit/eaa6d063374dec35aed3b3b12442f4c797702dc6))
* enable Traefik /ping endpoint for probes ([0d91be7](https://github.com/mikecsmith/http-template/commit/0d91be773bf264f538df1e4d29f886719acbb023))
* keep dist/dev/server visible to Tilt and seed dev binary at Tiltfile load ([a7e3aba](https://github.com/mikecsmith/http-template/commit/a7e3aba2d564271311df6ab8457dcc71e7adcedc))
* remove data wrapper from 200 json responses ([ac97c3c](https://github.com/mikecsmith/http-template/commit/ac97c3c9a2e75e95446dc0b5ff37e63872e39373))
* Tilt build and update mechanism via docker_build_with_restart ([b31c7e1](https://github.com/mikecsmith/http-template/commit/b31c7e10eeb426ee4d066dc4fb14080c5f418a74))
* warn logging instead of error on respond write issues ([fadbae9](https://github.com/mikecsmith/http-template/commit/fadbae95d6a79462824695d4c46c63e25adcf4fa))
* wrong error messages from DecodeValid example ([716fd07](https://github.com/mikecsmith/http-template/commit/716fd07b35275f789a8a24536f0d5130793653c4))
