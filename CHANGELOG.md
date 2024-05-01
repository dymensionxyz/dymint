# [](https://github.com/dymensionxyz/dymint/compare/v1.1.0-rc02...v) (2024-05-01)


### Bug Fixes

* **celestia test:** fix race in test ([#755](https://github.com/dymensionxyz/dymint/issues/755)) ([0b36781](https://github.com/dymensionxyz/dymint/commit/0b367818bf6aa8da4a4fd8e4e5c78223b60b44e0))
* **celestia:** impl retry on submit ([#748](https://github.com/dymensionxyz/dymint/issues/748)) ([61630eb](https://github.com/dymensionxyz/dymint/commit/61630eb458197abe2440a81426210000dff25d40))
* **da:** fixed da path seperator and encoding issue ([#731](https://github.com/dymensionxyz/dymint/issues/731)) ([3a3b219](https://github.com/dymensionxyz/dymint/commit/3a3b21932750fee7eaaa9c186f78e36e3e597746))
* **DA:** use expo backoff in retries ([#739](https://github.com/dymensionxyz/dymint/issues/739)) ([848085f](https://github.com/dymensionxyz/dymint/commit/848085f70bcaae81fb80da3ab78c4d8b399e13b1))
* **logging:** added reason for websocket closed debug msg ([#746](https://github.com/dymensionxyz/dymint/issues/746)) ([3aa7d80](https://github.com/dymensionxyz/dymint/commit/3aa7d80ace92b3b0f79e4f338f10bb94c96ab6dd))
* **logs:** make logs more readable in a couple places, fix race cond ([#749](https://github.com/dymensionxyz/dymint/issues/749)) ([f05ef39](https://github.com/dymensionxyz/dymint/commit/f05ef3957b754c05fbc90aa39eabce80bbe65933))
* **p2p:** validate block before applying and not before caching in p2p gossiping ([#723](https://github.com/dymensionxyz/dymint/issues/723)) ([98371b5](https://github.com/dymensionxyz/dymint/commit/98371b5220613e70f3274fab5593e02ba532f7db))
* **produce loop:** handle unauthenticated error in settlement layer ([#726](https://github.com/dymensionxyz/dymint/issues/726)) ([33e78d1](https://github.com/dymensionxyz/dymint/commit/33e78d116b5f14b91b8b3bda2b6cbfee9040e2d3))
* **rpc:** nil panic in rpc/json/handler.go WriteError ([#750](https://github.com/dymensionxyz/dymint/issues/750)) ([e09709b](https://github.com/dymensionxyz/dymint/commit/e09709b428a33da002defb9f13178fa19b81a69b))



# [1.1.0-rc02](https://github.com/dymensionxyz/dymint/compare/v1.1.0-rc01...v1.1.0-rc02) (2024-04-26)



# [1.1.0-rc01](https://github.com/dymensionxyz/dymint/compare/v1.0.1-alpha...v1.1.0-rc01) (2024-04-25)


### Bug Fixes

* **block production:** apply block before gossiping ([#695](https://github.com/dymensionxyz/dymint/issues/695)) ([5c496b4](https://github.com/dymensionxyz/dymint/commit/5c496b453e98bbcc67feb6df3a2d4ad340586816))
* **block:** Only register `nodeHealthStatusHandler` for sequencer ([#683](https://github.com/dymensionxyz/dymint/issues/683)) ([da2ff94](https://github.com/dymensionxyz/dymint/commit/da2ff94bcdd064109da703fa885609846a94e180))
* celestia wrong error log for availability checks and retrievals ([#646](https://github.com/dymensionxyz/dymint/issues/646)) ([eb08e30](https://github.com/dymensionxyz/dymint/commit/eb08e30da2c5d807db80d44237418e64be94abf0))
* **cmd:** Check if config.Genesis is provided on node start ([#681](https://github.com/dymensionxyz/dymint/issues/681)) ([4e1d383](https://github.com/dymensionxyz/dymint/commit/4e1d383254a9fb3a5856e33bfafebe203b60df91))
* **code standards:** a few wrongly formatted error objects ([#689](https://github.com/dymensionxyz/dymint/issues/689)) ([b4921cb](https://github.com/dymensionxyz/dymint/commit/b4921cbed57df1d74afe3bfaf10b9372c0f2b57f))
* **code standards:** fmt and fumpt ([#685](https://github.com/dymensionxyz/dymint/issues/685)) ([af50813](https://github.com/dymensionxyz/dymint/commit/af508133afe7acd0db5f4abaea9ea79720fbcbad))
* **code standards:** remove many 'failed to...' ([#686](https://github.com/dymensionxyz/dymint/issues/686)) ([e9d069e](https://github.com/dymensionxyz/dymint/commit/e9d069e3b41cdde724b2b986f4d5a03a7670c3e1))
* **code standards:** remove several error strings ([#688](https://github.com/dymensionxyz/dymint/issues/688)) ([2043c7f](https://github.com/dymensionxyz/dymint/commit/2043c7f5833ac7931aa91d28b849eaf403c9765b))
* **code standards:** use errors.Is in several places ([#687](https://github.com/dymensionxyz/dymint/issues/687)) ([f340371](https://github.com/dymensionxyz/dymint/commit/f340371dccd3033f58ddb0424766c182e31badc9))
* **concurrency:** applying blocks concurrently can lead to unexpected errors ([#700](https://github.com/dymensionxyz/dymint/issues/700)) ([7290af6](https://github.com/dymensionxyz/dymint/commit/7290af6f00887d09862f32bf43169f42a416e87e))
* **concurrency:** remove unused sync cond ([#706](https://github.com/dymensionxyz/dymint/issues/706)) ([5f325e5](https://github.com/dymensionxyz/dymint/commit/5f325e5dd0e6b607b7bbc399a034856f47a9e305))
* **concurrency:** use atomic specific types instead of atomic helpers ([#682](https://github.com/dymensionxyz/dymint/issues/682)) ([1628a5c](https://github.com/dymensionxyz/dymint/commit/1628a5c23569b24b2ac11494eb90e79b4e2797cd))
* **config:** Add missing config validation ([#679](https://github.com/dymensionxyz/dymint/issues/679)) ([cfaa8ce](https://github.com/dymensionxyz/dymint/commit/cfaa8cefdf6d8f9ede7ac92c9a37d450c483e156))
* **da:** check DA client type before trying to fetch batch ([#714](https://github.com/dymensionxyz/dymint/issues/714)) ([7180587](https://github.com/dymensionxyz/dymint/commit/718058755a394bf6652b248f792013c11e751d64))
* **da:** full-node get the da fetch configuration from hub and not config ([#719](https://github.com/dymensionxyz/dymint/issues/719)) ([6bc6c97](https://github.com/dymensionxyz/dymint/commit/6bc6c97cb7f9b5d2cf9587f0148261fa3ef7446f))
* full-node panics with app hash mismatch error when syncing ([#647](https://github.com/dymensionxyz/dymint/issues/647)) ([0073faf](https://github.com/dymensionxyz/dymint/commit/0073faf8608af7d1c74985c188b0ca540488ab9d))
* **gossip:** validate blocks when receiving them over gossip ([#699](https://github.com/dymensionxyz/dymint/issues/699)) ([18f98d2](https://github.com/dymensionxyz/dymint/commit/18f98d23ba9eb33dc7092fff446d4a733a2d36a2))
* **manager:** more robust error handling and health status ([#696](https://github.com/dymensionxyz/dymint/issues/696)) ([ab41f13](https://github.com/dymensionxyz/dymint/commit/ab41f137ec25139470f333f3446c4ba46919309f))
* **manager:** re-use submitted DA batches on SL failure ([#708](https://github.com/dymensionxyz/dymint/issues/708)) ([d71f4e2](https://github.com/dymensionxyz/dymint/commit/d71f4e2eb4af45a2d676f799079e7a14503b4604))
* **manager:** use mutex instead of atomic batchInProcess ([#678](https://github.com/dymensionxyz/dymint/issues/678)) ([2ccbbd6](https://github.com/dymensionxyz/dymint/commit/2ccbbd65290b1fdc0003420b734078953de9d190))
* **mempool:** Initialize the mempool with the correct(current) block height ([#703](https://github.com/dymensionxyz/dymint/issues/703)) ([35b9e58](https://github.com/dymensionxyz/dymint/commit/35b9e588c9b657b47c1b3379a65956b38cbe7219))
* **mempool:** set pre and post tx check funcs after genesis, not only after first block ([#691](https://github.com/dymensionxyz/dymint/issues/691)) ([849ba80](https://github.com/dymensionxyz/dymint/commit/849ba80900e270ef455a2ae583c31a21f5751990))
* **p2p:** handle default mempool check tx error case ([#698](https://github.com/dymensionxyz/dymint/issues/698)) ([fb0d547](https://github.com/dymensionxyz/dymint/commit/fb0d5475e61f3b1f3fdd7474d2c5de604ee35558))
* remove hub dependency importing pseudo version of osmosis ([#627](https://github.com/dymensionxyz/dymint/issues/627)) ([d609ad8](https://github.com/dymensionxyz/dymint/commit/d609ad8b1d96673918fc157ce35e9c6f8fbe68c1))
* remove occurrence of Tendermint string ([#676](https://github.com/dymensionxyz/dymint/issues/676)) ([01ff0a4](https://github.com/dymensionxyz/dymint/commit/01ff0a496ddda5df264e48e664c779036175a109))
* retries for hub client ([#630](https://github.com/dymensionxyz/dymint/issues/630)) ([48bc6bf](https://github.com/dymensionxyz/dymint/commit/48bc6bfdd167dd5b9e5a8b7fea5784556e2d9c1d))
* **rpc:** Added Timeout for RPC handler ([#673](https://github.com/dymensionxyz/dymint/issues/673)) ([cefca7a](https://github.com/dymensionxyz/dymint/commit/cefca7aaa59d749b54d294a63be72bf8ab9b74ea))
* **rpc:** broken ws upgrade introduced by using http.TimeoutHandler ([#702](https://github.com/dymensionxyz/dymint/issues/702)) ([a8f5f9c](https://github.com/dymensionxyz/dymint/commit/a8f5f9cd7575b65ed83d21d9ca9242b42ae25e4d))
* **rpc:** set const ReadHeaderTimeout to dymint RPC server ([#671](https://github.com/dymensionxyz/dymint/issues/671)) ([4c05a1d](https://github.com/dymensionxyz/dymint/commit/4c05a1db448768c73f85ec8b847bf05adbbb6192))
* sequencer stops posting batches after hub disconnection ([#626](https://github.com/dymensionxyz/dymint/issues/626)) ([9733e1c](https://github.com/dymensionxyz/dymint/commit/9733e1c03b7bb0b8d4cd4e1b0238f31c301a726f))
* set default empty block generation time to 1 hour ([#623](https://github.com/dymensionxyz/dymint/issues/623)) ([1ba1a39](https://github.com/dymensionxyz/dymint/commit/1ba1a39dec16a95f59346fe186a9cf8e7d08ec17))
* **settlement:** removed deprecated `IntermediateStateRoot` from BD before batch submission ([#642](https://github.com/dymensionxyz/dymint/issues/642)) ([3cd56c5](https://github.com/dymensionxyz/dymint/commit/3cd56c57c597282d79af80644b1778d80420befa))
* **store:** do not discard CAS fail errors ([#705](https://github.com/dymensionxyz/dymint/issues/705)) ([3bcda30](https://github.com/dymensionxyz/dymint/commit/3bcda306203e9e7ec6cf65df70b8086343631ded))
* updated go version and x/net library ([#670](https://github.com/dymensionxyz/dymint/issues/670)) ([b766728](https://github.com/dymensionxyz/dymint/commit/b76672877574cbda3d41961f9b6bcb52a47a8460))


### Features

* **ci:** Add changelog workflow ([#720](https://github.com/dymensionxyz/dymint/issues/720)) ([6361f97](https://github.com/dymensionxyz/dymint/commit/6361f974c5b51f4d6339737812c30b3adc8be980))
* Enforce config rollapp id to be same as genesis chain id ([#697](https://github.com/dymensionxyz/dymint/issues/697)) ([84e8853](https://github.com/dymensionxyz/dymint/commit/84e885371418fb16ca9f89ebd2be613e68588d7e))
* **mempool:** add a sanity check ([#690](https://github.com/dymensionxyz/dymint/issues/690)) ([c4379ff](https://github.com/dymensionxyz/dymint/commit/c4379ff97d0e2d2bab5726194528af602904b819))
* **perf:** removed unneeded state-index query ([#650](https://github.com/dymensionxyz/dymint/issues/650)) ([25afe20](https://github.com/dymensionxyz/dymint/commit/25afe20fea420b930a8a221cc4c78620b0a7b510))



## [1.0.1-alpha](https://github.com/dymensionxyz/dymint/compare/v1.0.0-alpha...v1.0.1-alpha) (2024-03-26)



# [1.0.0-alpha](https://github.com/dymensionxyz/dymint/compare/v0.6.1-beta...v1.0.0-alpha) (2024-03-21)


### Bug Fixes

* changing dymint rpc to enable queries for unhealthy nodes ([#592](https://github.com/dymensionxyz/dymint/issues/592)) ([da7f62c](https://github.com/dymensionxyz/dymint/commit/da7f62c4aee642760910540147a2cf739f9479d6))
* fix local da setup ([#600](https://github.com/dymensionxyz/dymint/issues/600)) ([8cd1a10](https://github.com/dymensionxyz/dymint/commit/8cd1a1092f93e2d92442978a58f34e00f4b9f4d9))
* skip trying to apply blocks from DA with mismatched heights. ([#594](https://github.com/dymensionxyz/dymint/issues/594)) ([d2a0a97](https://github.com/dymensionxyz/dymint/commit/d2a0a97955134690dae4224b0b7c3207f9283c96))
* wrong submitted dapath to dimension hub. client info missing [#586](https://github.com/dymensionxyz/dymint/issues/586) ([#587](https://github.com/dymensionxyz/dymint/issues/587)) ([06c528d](https://github.com/dymensionxyz/dymint/commit/06c528d60b4608d2d8bb9a1d0de0ff915a85cfc2))


### Features

* Add data availability validation for Celestia DA [#422](https://github.com/dymensionxyz/dymint/issues/422) ([#575](https://github.com/dymensionxyz/dymint/issues/575)) ([f7254f4](https://github.com/dymensionxyz/dymint/commit/f7254f488f4bc99aecf84a4d7dfefccbf2a88dd0))
* Enable shared mock settlement ([#549](https://github.com/dymensionxyz/dymint/issues/549)) ([996e681](https://github.com/dymensionxyz/dymint/commit/996e681cb937b22b4fba9d159539ccee66de4b7e))
* reconnect full node to sequencer with configurable time check ([#556](https://github.com/dymensionxyz/dymint/issues/556)) ([9129983](https://github.com/dymensionxyz/dymint/commit/912998300733bae1f60ae42a63b07ea65cfc17bc))



## [0.6.1-beta](https://github.com/dymensionxyz/dymint/compare/v0.6.0-beta...v0.6.1-beta) (2024-01-01)


### Bug Fixes

* avail submission was stuck waiting for finalization ([#438](https://github.com/dymensionxyz/dymint/issues/438)) ([aa1e967](https://github.com/dymensionxyz/dymint/commit/aa1e96747cbb93137ca2c664d83bed04592ba2e7))
* checking DA received batches ([#527](https://github.com/dymensionxyz/dymint/issues/527)) ([d2b2cdf](https://github.com/dymensionxyz/dymint/commit/d2b2cdfb13ae4ae5124819909c0d35229cd73102))
* Display actual version in RPC status ([#488](https://github.com/dymensionxyz/dymint/issues/488)) ([5b5dcdb](https://github.com/dymensionxyz/dymint/commit/5b5dcdb35fe8e1c19f9e5476d5b8e953951f84c1))
* dymint fails to submit new batches after sl failure ([#435](https://github.com/dymensionxyz/dymint/issues/435)) ([11a5c2b](https://github.com/dymensionxyz/dymint/commit/11a5c2b4fb7c2ffb7c6bbbaaeef7726afda83a8d))
* dymint out of sync with hub on tx submission timeout ([#404](https://github.com/dymensionxyz/dymint/issues/404)) ([87343ed](https://github.com/dymensionxyz/dymint/commit/87343ed7c644e0cc18567585037bbed64068cc8c))
* dynamic subscriber name to avoid possible subscriber collision ([#442](https://github.com/dymensionxyz/dymint/issues/442)) ([1bc4571](https://github.com/dymensionxyz/dymint/commit/1bc4571beb36d83e0b243758dee6d0b468b39de1))
* exponantial timeout when submitting to the hub ([#458](https://github.com/dymensionxyz/dymint/issues/458)) ([40e74dc](https://github.com/dymensionxyz/dymint/commit/40e74dca8a13df87020c7c426348a786e909a91c))
* fix avail finalization timeout ([#456](https://github.com/dymensionxyz/dymint/issues/456)) ([3e1ed62](https://github.com/dymensionxyz/dymint/commit/3e1ed6250b4aa50ab931381cb8ff9a031005464c))
* fix gas adjustment parameter ([#432](https://github.com/dymensionxyz/dymint/issues/432)) ([9ce91a0](https://github.com/dymensionxyz/dymint/commit/9ce91a0eba3fd838c7c0934dd5bddeccc71e9e27))
* fixed avail in toml ([#407](https://github.com/dymensionxyz/dymint/issues/407)) ([ff2fc27](https://github.com/dymensionxyz/dymint/commit/ff2fc27796456b516398fc75d3a064eec097603a))
* Fixed bug in test where we didn't wait for settlement client to stop ([#415](https://github.com/dymensionxyz/dymint/issues/415)) ([d690b09](https://github.com/dymensionxyz/dymint/commit/d690b09a36211452a32d7279541185b96e613990))
* fixed bug where the unhealthy da event didn't stop block production ([#431](https://github.com/dymensionxyz/dymint/issues/431)) ([3746df2](https://github.com/dymensionxyz/dymint/commit/3746df2e34332d4cad6f3b57b10f26f1738990e0))
* fixed configuration validation, to support empty_blocks=0s ([#444](https://github.com/dymensionxyz/dymint/issues/444)) ([5aa1f5f](https://github.com/dymensionxyz/dymint/commit/5aa1f5f9ca5e5a34720f67ce62f7f72bce77a16c))
* fixed race condition in some DA tests ([#447](https://github.com/dymensionxyz/dymint/issues/447)) ([c25b4e1](https://github.com/dymensionxyz/dymint/commit/c25b4e1000678ad9aba5ca69c5a94cb1cf95e646))
* initializing LastValidatorSet as well on InitChain ([#390](https://github.com/dymensionxyz/dymint/issues/390)) ([93642b5](https://github.com/dymensionxyz/dymint/commit/93642b5fbde98a5846b6afec7ac2d462fcb8e995))
* nodes keep out of sync when missing gossiped block ([#540](https://github.com/dymensionxyz/dymint/issues/540)) ([14ae6fd](https://github.com/dymensionxyz/dymint/commit/14ae6fd02cb696cbcf35d67931ed2769b892404a))
* possible race condition with small batches upon batch submission  ([#410](https://github.com/dymensionxyz/dymint/issues/410)) ([9feaced](https://github.com/dymensionxyz/dymint/commit/9feaced004a63c98d10a1cd89571a364da413f5f))
* reduced empty blocks submission ([#452](https://github.com/dymensionxyz/dymint/issues/452)) ([2e9bb1d](https://github.com/dymensionxyz/dymint/commit/2e9bb1dae9acb58ca9eb20aeec8fbc6134b407a8))
* updated go-cnc to match celetia light node v0.11 ([#400](https://github.com/dymensionxyz/dymint/issues/400)) ([a67e445](https://github.com/dymensionxyz/dymint/commit/a67e445a19d52fcf523fee86cec5f76957767e0c))


### Features

* add basic rollapp Metrics for rollapp height and hub height  ([#420](https://github.com/dymensionxyz/dymint/issues/420)) ([190379d](https://github.com/dymensionxyz/dymint/commit/190379d54ea483b4b5299b6e17592524cb2dbac0))
* Add Prometheus HTTP Metric Server for Enhanced Monitoring ([#419](https://github.com/dymensionxyz/dymint/issues/419)) ([479bfb8](https://github.com/dymensionxyz/dymint/commit/479bfb824a955a3bbdd2db53580cef10e6a97c72))
* celestia fee should be calculated dynamically ([#417](https://github.com/dymensionxyz/dymint/issues/417)) ([e5a48aa](https://github.com/dymensionxyz/dymint/commit/e5a48aafdb1bbac4f899c2ace88f80fe515a01b0))
* gas adjustmnet parsable for celestia config ([#425](https://github.com/dymensionxyz/dymint/issues/425)) ([91cebf7](https://github.com/dymensionxyz/dymint/commit/91cebf7e854880b560fa46510539c66047cc667c))
* tendermint headers compatibility  ([#505](https://github.com/dymensionxyz/dymint/issues/505)) ([ec633de](https://github.com/dymensionxyz/dymint/commit/ec633de8b9977ca454ea741c47bf33c2ea0a9486))
* try to append empty block at the end of each non-empty batch ([#472](https://github.com/dymensionxyz/dymint/issues/472)) ([fbb47c9](https://github.com/dymensionxyz/dymint/commit/fbb47c9648c66e10241b3db955bbf056698a3083))
* update celestia fee calculation function ([#427](https://github.com/dymensionxyz/dymint/issues/427)) ([6063981](https://github.com/dymensionxyz/dymint/commit/6063981924aeed343693baf27b2b6a3e03f6f1a9))



# [0.5.0-rc1](https://github.com/dymensionxyz/dymint/compare/v0.4.0-beta...v0.5.0-rc1) (2023-06-29)


### Bug Fixes

* celesita tx batch timeout doesn't poll for inclusion  ([#339](https://github.com/dymensionxyz/dymint/issues/339)) ([dba8489](https://github.com/dymensionxyz/dymint/commit/dba8489214051da290dcd841c1ca3dbdf61b8dfd))
* distinguish between different errors of state loading ([#345](https://github.com/dymensionxyz/dymint/issues/345)) ([3e1f6eb](https://github.com/dymensionxyz/dymint/commit/3e1f6eb9890563f5b2a20ad9375c1ebd964b84af))
* dymint out of sync in case of missed hub state update event ([#384](https://github.com/dymensionxyz/dymint/issues/384)) ([6c190da](https://github.com/dymensionxyz/dymint/commit/6c190dad28803dc67c9f34a4928af98b8fa28258))
* fixed bug where health event from settlement layer were ignored ([#385](https://github.com/dymensionxyz/dymint/issues/385)) ([73a177d](https://github.com/dymensionxyz/dymint/commit/73a177d4574bb3bc4c4eaf4354cf74bb2af90918))
* fixed namespaceID parsing from toml ([#373](https://github.com/dymensionxyz/dymint/issues/373)) ([b38719a](https://github.com/dymensionxyz/dymint/commit/b38719a4f0916cf658834d5a196e2b34d4887c32))
* fixed parsing rollappID parameter ([#379](https://github.com/dymensionxyz/dymint/issues/379)) ([a6e92dc](https://github.com/dymensionxyz/dymint/commit/a6e92dc0b1cda7df5a4c513daf4ca38487b5639e))
* pre commit false negative on local machine ([#322](https://github.com/dymensionxyz/dymint/issues/322)) ([9463179](https://github.com/dymensionxyz/dymint/commit/9463179c6bb8100a5870cbffe6fff7dad361005c))
* removed panic alert from graceful shutdown ([#331](https://github.com/dymensionxyz/dymint/issues/331)) ([bbd9f01](https://github.com/dymensionxyz/dymint/commit/bbd9f0102bb936e38ace0751bdfb17ee36fd42b6))
* returns descriptive error in case no sequencer registered on the hub ([#305](https://github.com/dymensionxyz/dymint/issues/305)) ([1f3cc05](https://github.com/dymensionxyz/dymint/commit/1f3cc05a14ec7e14cbcd35cddb0de353786052a4))
* rollapp evm on devnet crashing with lastresulthash mismatch ([#375](https://github.com/dymensionxyz/dymint/issues/375)) ([e22c252](https://github.com/dymensionxyz/dymint/commit/e22c252499d295e910194db7df17779ad3cdb30f))
* skipping nil txs in the search result ([#346](https://github.com/dymensionxyz/dymint/issues/346)) ([3319145](https://github.com/dymensionxyz/dymint/commit/3319145661c14baf6b4e0595d5d7650ca018ab58))


### Features

* add pruning mechanism that deletes old blocks and commits ([#328](https://github.com/dymensionxyz/dymint/issues/328)) ([72e5acf](https://github.com/dymensionxyz/dymint/commit/72e5acf37eead9e0e6276ad1639fb338c3643953))
* Add support for avail as a DA ([#355](https://github.com/dymensionxyz/dymint/issues/355)) ([1a683ca](https://github.com/dymensionxyz/dymint/commit/1a683caee04c82c0b79a0fe981c7c35a993cc00a))
* added max bytes size check when creating the batch ([#321](https://github.com/dymensionxyz/dymint/issues/321)) ([8b006de](https://github.com/dymensionxyz/dymint/commit/8b006defb6f15e56f42013fa408cda91712a9732))
* added toml parser for configuration ([#358](https://github.com/dymensionxyz/dymint/issues/358)) ([d31da1a](https://github.com/dymensionxyz/dymint/commit/d31da1ac75b38bc36b5d8574a5f67a2d9065282b))
* better enforcement on dymint flags ([#377](https://github.com/dymensionxyz/dymint/issues/377)) ([cc439f3](https://github.com/dymensionxyz/dymint/commit/cc439f36ffada611a67830ee02b88bc95eb169a6))
* mock SL doesn't use keyring path from config.  ([#367](https://github.com/dymensionxyz/dymint/issues/367)) ([5abf3ca](https://github.com/dymensionxyz/dymint/commit/5abf3ca9dece8132cc4dd2d17f6baef5afd4b7af))
* propagate node health to RPC server response ([#327](https://github.com/dymensionxyz/dymint/issues/327)) ([5615c11](https://github.com/dymensionxyz/dymint/commit/5615c11a93de104ca62eef6ad9cb01b228244808))
* refactor dymint config to support loading configuration from file ([#326](https://github.com/dymensionxyz/dymint/issues/326)) ([b59ca6f](https://github.com/dymensionxyz/dymint/commit/b59ca6f24577101a1ba60f7a16e145c824b1f01e))
* stop chain block production if node unhealthy event is emitted ([#319](https://github.com/dymensionxyz/dymint/issues/319)) ([7b017fc](https://github.com/dymensionxyz/dymint/commit/7b017fc130c37f8297d2931dc9881f2bb0f94f77))
* support skipping empty blocks production ([#342](https://github.com/dymensionxyz/dymint/issues/342)) ([09cab6a](https://github.com/dymensionxyz/dymint/commit/09cab6a5417f3ab98434704d91fa6167df78aa28))
* toggle node health status based on data batch submission status ([#341](https://github.com/dymensionxyz/dymint/issues/341)) ([5ffd07d](https://github.com/dymensionxyz/dymint/commit/5ffd07d0eab5a759fa863d7211b8d9aaff08db75))
* toggle settlement health event upon settlement batch submission status ([#332](https://github.com/dymensionxyz/dymint/issues/332)) ([7c65250](https://github.com/dymensionxyz/dymint/commit/7c65250f2a355a540bf2bc3e52981609332490d9))



# [0.4.0-beta](https://github.com/dymensionxyz/dymint/compare/v0.3.1-beta...v0.4.0-beta) (2023-04-23)


### Bug Fixes

* bad format of rpc subscribe response  ([#278](https://github.com/dymensionxyz/dymint/issues/278)) ([689eabe](https://github.com/dymensionxyz/dymint/commit/689eabe95a650ad9315411356d07436c16c4d2b9))
* changed websocket connection to allow CORS based on the middleware ([#268](https://github.com/dymensionxyz/dymint/issues/268)) ([6151a3c](https://github.com/dymensionxyz/dymint/commit/6151a3c9d4daa3e5db2ab09bdb5d557519f9cf1f))
* dymint uses uncapped backoff delay for submitting batches ([#304](https://github.com/dymensionxyz/dymint/issues/304)) ([c217bda](https://github.com/dymensionxyz/dymint/commit/c217bda963afb505e6addcd46ced838f84d9ae6f))
* fixed the unmarshall of array to an object on websocket ([#274](https://github.com/dymensionxyz/dymint/issues/274)) ([9522808](https://github.com/dymensionxyz/dymint/commit/95228081927facbc6ec08a82a2458e6fa020a394))
* handling queries with page=0 argument ([#264](https://github.com/dymensionxyz/dymint/issues/264)) ([0f649bf](https://github.com/dymensionxyz/dymint/commit/0f649bfef9dc3696cc7fd09b505aabeaa7cee5a0))
* inconsistent height caused due to crash during commit  ([#266](https://github.com/dymensionxyz/dymint/issues/266)) ([5758987](https://github.com/dymensionxyz/dymint/commit/575898748642708e0ec314818d0f344ca7e27448))
* terminate stateUpdateHandler upon subscription termination ([#295](https://github.com/dymensionxyz/dymint/issues/295)) ([162b3e6](https://github.com/dymensionxyz/dymint/commit/162b3e66d35edb2a6aff83834c21b8483f61acf1))



## [0.3.1-beta](https://github.com/dymensionxyz/dymint/compare/v0.3.0-beta...v0.3.1-beta) (2023-02-19)


### Bug Fixes

* changed latestHeight to be updated on batch acceptance  ([#257](https://github.com/dymensionxyz/dymint/issues/257)) ([b3642ce](https://github.com/dymensionxyz/dymint/commit/b3642ce4fc790065446dcca30a1715765ae250e1))
* submit batch keep retrying even though batch accepted event received ([#253](https://github.com/dymensionxyz/dymint/issues/253)) ([e6f489c](https://github.com/dymensionxyz/dymint/commit/e6f489c7ab5566d71cba5be321460cfec11cd8c0))



# [0.3.0-beta](https://github.com/dymensionxyz/dymint/compare/v0.1.1-alpha...v0.3.0-beta) (2023-02-15)


### Features

* Updated dymension client to accept gas limit, fee and prices params. ([#243](https://github.com/dymensionxyz/dymint/issues/243)) ([a738e1d](https://github.com/dymensionxyz/dymint/commit/a738e1d8ac68d8df6e9a5eabe72992c5a3dab3b1))



## [0.1.1-alpha](https://github.com/dymensionxyz/dymint/compare/v0.1.0-alpha...v0.1.1-alpha) (2022-11-28)



# [0.1.0-alpha](https://github.com/dymensionxyz/dymint/compare/v0.3.4...v0.1.0-alpha) (2022-09-19)


### Features

* sync from events vs optimistically ([#84](https://github.com/dymensionxyz/dymint/issues/84)) ([ad66d93](https://github.com/dymensionxyz/dymint/commit/ad66d93a1427af2b711339e9e8112c572d8a7087))
* sync state from settlement layer state update events ([#79](https://github.com/dymensionxyz/dymint/issues/79)) ([66b9465](https://github.com/dymensionxyz/dymint/commit/66b94658b341406393dcc96745606147b1fd9a6d))



## [0.3.4](https://github.com/dymensionxyz/dymint/compare/v0.3.3...v0.3.4) (2022-07-05)


### Bug Fixes

* change log level of WS message type error to debug ([d876852](https://github.com/dymensionxyz/dymint/commit/d876852f9a9807197825a139456701b2b91902d3))
* ensure JSON serialization compatibility ([e454904](https://github.com/dymensionxyz/dymint/commit/e4549046b57cce680fb2fc495112b663efedaad1)), closes [#463](https://github.com/dymensionxyz/dymint/issues/463)
* use Tendermint JSON serlization for JSON RPC results ([27d3eae](https://github.com/dymensionxyz/dymint/commit/27d3eae7441a96fa870030c7fcc7e276287eb507))



## [0.3.3](https://github.com/dymensionxyz/dymint/compare/v0.3.2...v0.3.3) (2022-06-21)


### Bug Fixes

* handle ConsensusParams in state deserialization ([05d6e67](https://github.com/dymensionxyz/dymint/commit/05d6e67aba9a7c3ab8c020f360c79b6ff9780c05))
* improve error handling in WebSockets ([f2748ae](https://github.com/dymensionxyz/dymint/commit/f2748aea82145a603c7a0a7b9d5fca1955982d9b))
* pass logger to WS connection handle ([ade5622](https://github.com/dymensionxyz/dymint/commit/ade56221ff1616f487b034ae790e354d03ad0b40))


### Features

* improved block submission error handling ([bd949ca](https://github.com/dymensionxyz/dymint/commit/bd949ca5e43cd61e88a525b3b48f3d16b8520bd1))
* serialize state with protobuf ([#424](https://github.com/dymensionxyz/dymint/issues/424)) ([3c4318f](https://github.com/dymensionxyz/dymint/commit/3c4318f9cf9047a54405a86e9ed20a99813944c3))



## [0.3.1](https://github.com/dymensionxyz/dymint/compare/v0.3.0...v0.3.1) (2022-06-01)


### Bug Fixes

* ensure Code field in TxResponse has correct value ([d4ec3a5](https://github.com/dymensionxyz/dymint/commit/d4ec3a5441b44dd00c32f73dbbb9dd8ca1e5bbf9))
* handle application level errors in celestia-node API ([214cd82](https://github.com/dymensionxyz/dymint/commit/214cd82014c350ff782ef1c09f93876631058a22))


### Features

* use go-cnc v0.1.0 to replace libs/cnrc ([5055a87](https://github.com/dymensionxyz/dymint/commit/5055a87448d53d98982e5b4b77f5573f35d31c7e))



# [0.3.0](https://github.com/dymensionxyz/dymint/compare/v0.2.0...v0.3.0) (2022-05-25)


### Bug Fixes

* actually use new config options ([c5a9a53](https://github.com/dymensionxyz/dymint/commit/c5a9a5379e84d901368878cb496ddef5d6d8b073))
* add GRPC plugin to protobuf generation ([c1b448a](https://github.com/dymensionxyz/dymint/commit/c1b448a437c4b022a8023091cc3d5c782439ef7c))
* add missing configuration options ([a3e3988](https://github.com/dymensionxyz/dymint/commit/a3e39880343f5dab4f80864ab03ab5e853afbe21))
* keep track of DA height of latest *applied* block ([e71c361](https://github.com/dymensionxyz/dymint/commit/e71c3619feee8677e18501436a622d0df850936c))
* make it work ([16933e5](https://github.com/dymensionxyz/dymint/commit/16933e543c59df785475cc60864ffaf9e7dcddc9))


### Features

* build proto deterministically with docker ([7784025](https://github.com/dymensionxyz/dymint/commit/77840254fc025cc1d593fd0abd8c1023f7e29390))
* Celestia DA Layer Client implementation ([#399](https://github.com/dymensionxyz/dymint/issues/399)) ([0848bfe](https://github.com/dymensionxyz/dymint/commit/0848bfe60b0d7bca7b08df37c1fd5c6e3530d845))
* retry DA submission ([375550b](https://github.com/dymensionxyz/dymint/commit/375550ba3990ded1c1b0652dddeec73bb66d1912))
* updated protobuf types ([6340945](https://github.com/dymensionxyz/dymint/commit/6340945d28b74438c7cb3a3fb1008b82bdde8141))



# [0.2.0](https://github.com/dymensionxyz/dymint/compare/v0.1.1...v0.2.0) (2022-04-11)


### Bug Fixes

* address review comments ([eb2e87c](https://github.com/dymensionxyz/dymint/commit/eb2e87c1193c77464440b29ee89f93dd7298363e))
* execute evmos build only once per PR ([47f9e4e](https://github.com/dymensionxyz/dymint/commit/47f9e4e45c1e33eecd52aba834a683435cb3bf5c))
* goimports files ([5ff1ddd](https://github.com/dymensionxyz/dymint/commit/5ff1ddd932d11b60b3ff611c06b1ccbf2c719dcb))
* make test logger actually threadsafe ([8a758cb](https://github.com/dymensionxyz/dymint/commit/8a758cb0c12af8e7ecda0600ba368e66f6f2affe))
* markdownlint -fix . ([ce81488](https://github.com/dymensionxyz/dymint/commit/ce814886083cf29c6be5c109b4c1a840ba1600fb))


### Features

* add DAStartHeight configuration to block.Manager ([3e4a966](https://github.com/dymensionxyz/dymint/commit/3e4a966503ba923b392e860c6cdf2c4510adcdfa))
* allow hard tabs in code blocks in md files ([4280b44](https://github.com/dymensionxyz/dymint/commit/4280b440f7f180b7831b9128a0d08524bda01a5d))
* disable MD013 (line length) check ([2a063f4](https://github.com/dymensionxyz/dymint/commit/2a063f417d1ca7565c1dd7eee83719d4dfcfc19f))
* implement new DALC API ([4db8cc2](https://github.com/dymensionxyz/dymint/commit/4db8cc2a6ad05ee557dc1270ceafa11e5b0ecca3))
* improve mock DA implementation ([f2395d7](https://github.com/dymensionxyz/dymint/commit/f2395d7ce6ce5de85150ed443d85937ab1be26b2))
* improved mock DA implementation ([cf371b2](https://github.com/dymensionxyz/dymint/commit/cf371b23ffd7b3710514d840c1a4067a0fe8180d))
* new DALC proto definitions ([1879977](https://github.com/dymensionxyz/dymint/commit/187997772be900bb15e04fe2abda3157200a9869))
* set allow_different_nesting to true ([f1ae476](https://github.com/dymensionxyz/dymint/commit/f1ae4765f5e8d96f0798fdf8a32635fa663175d2))



## [0.1.1](https://github.com/dymensionxyz/dymint/compare/v0.1.0...v0.1.1) (2022-03-08)


### Bug Fixes

* make `TestValidatorSetHandling` even more stable ([#314](https://github.com/dymensionxyz/dymint/issues/314)) ([743746a](https://github.com/dymensionxyz/dymint/commit/743746a7dd3c6405f2cc85a79cf9e4d3bd80e180))



# [0.1.0](https://github.com/dymensionxyz/dymint/compare/feb7aab58358d4718b83ff6904a6cec95cac35a5...v0.1.0) (2022-03-07)


### Bug Fixes

* do save ABCI responses for blocks ([#285](https://github.com/dymensionxyz/dymint/issues/285)) ([12b4451](https://github.com/dymensionxyz/dymint/commit/12b445142c811a6d0a77b6f55e87ad1779edb91e))
* gofmt block/manager.go and remove typo ([#222](https://github.com/dymensionxyz/dymint/issues/222)) ([feb7aab](https://github.com/dymensionxyz/dymint/commit/feb7aab58358d4718b83ff6904a6cec95cac35a5))
* make `TestValidatorSetHandling` stable ([#313](https://github.com/dymensionxyz/dymint/issues/313)) ([989fb16](https://github.com/dymensionxyz/dymint/commit/989fb16d453eba95ff281602076b1d27e7f5b068))


### Features

* implement BlockResults RPC function ([#263](https://github.com/dymensionxyz/dymint/issues/263)) ([48d3f30](https://github.com/dymensionxyz/dymint/commit/48d3f30cccc088629ef914b330bf83334324311b))



