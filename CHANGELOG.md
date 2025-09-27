
<a name="0.15.1"></a>
## [0.15.1](https://git.mills.io/yarnsocial/yarn/compare/0.15.0...0.15.1) (2022-10-29)

### Bug Fixes

* Fix a cache map concurrent read/write bug

### Features

* Add Access-Control-Allow-Origin (CORS) header to all API endpoints (#1054)


<a name="0.15.0"></a>
## [0.15.0](https://git.mills.io/yarnsocial/yarn/compare/0.14.0...0.15.0) (2022-10-29)

### Bug Fixes

* Fix spelling errors
* Fix Change Log for v0.15
* Fix link opening preference and disallow HTML links from overriding the target attribute based on user preference (#1052)
* Fix FilterOutFeedsAndBotsFactory() to use previously cached Twters for feed metadata
* Fix Mute and Bookmark Btn Disapearing (#1048)
* Fix possibly security/privact flaw in processing Markdown that allowed artbitrary style on HTML elements (#1047)
* Fix security flaw in link parsing and only allow standard urls (#1046)
* Fix tests
* Fix rendering line breaks by NOT stripping <br /> tags from the rendered output during HTML Santization 🤦‍♂️
* Fix FormatTwt markdown handling so posts entirely in parens are actually rendered correctly (#1045)
* Fix position of the Remember Me checkbox o Login
* Fix missing fieldset and labels for register and login forms and improve some help tooltips (#1043)
* Fix behaviour of Open Registrations and /register handler so toggling Open Registrations and Join work seamlessly
* Fix formatting of Atom Feeds of Users and the Pod's Local Timeline (#1040)
* Fix behaviour of the link and image link buttons (#1039)
* Fix deleting/editing last Twt behaviour where delete would also post what's in the postbox (#1038)
* Fix typos (#1035)
* Fix renaming merged feed to avoid invalid cross-device link errors
* Fix missing aria-busy css (#1030)
* Fix and improve the language around Email login and help text (#1026)
* Fix missing aria-busy css (#1025)
* Fix the accent bg colour of amoled theme to improve contrast (#1023)
* Fix e2e tests by enabling open registrations
* Fix another templat ebug (wtf)
* Fix a template bug
* Fix amoled theme
* Fix API log messages
* Fix setting to show hostname of account to match profile view
* Fix registration spam (#981)
* Fix confusing behaviour of Reply button on Forked conversations in /conv/ view (#1001)
* Fix behaviour of Follow / Unfollow buttons (#1000)
* Fix preflight checks to support Go 1.19
* Fix indentation test
* Fix typo and remove trailing space
* Fix lightbox JS error on close (#948)

### Features

* Add completed release notes for v0.15
* Add Chnage Log for v0.15 and prepare for the Release Notes
* Add pager support to urlForRootConv (#1053)
* Add blockquote button to postbox (#1051)
* Add comments on the now badly implemented FilterOutFeedsAndBots FilterFunc
* Add FeatureFixMovedTwters (fix_moved_twters) to try to address bad cached Twters data for moved feeds (302)
* Add a new debug endpoint to dump the pod's cache
* Add Access-Control-Allow-Origin (CORS) Headers for Twtxt, Atom and Avatar endpoints (#1044)
* Add feedtype icons to profile views too
* Add feedtype icons next to the nick of each twt (#1036)
* Add Sync API and sync sub-command for yarnc for syncing a local feed to a Yarn.social pod (#1033)
* Add an improved version of the Settings view with Tabs (#1032)
* Add /api/v1/whoami endpoint that exposes the Username of the logged in user (#1028)
* Add Pod Logo resource endpoint and update Info resource to include link (#1029)
* Add a little Javascript (optional of course) to support Drag 'n Drop to Upload files/media (#1031)
* Add social.kyoko-project.wer.ee to known pods
* Add yn.vern.cc to known pods
* Add a /join endpoint and join page to combat spam and unwanted seo link submission crap until we build a proper invite system
* Add support for serving archived/rotated feeds and the prev Metadata field (#1017)
* Add Amoled color theme
* Add support for indexing and searching an archive (#1010)
* Add tool to validate feeds for analysis purposes (#1009)
* Add optional --log-file support (#1003)
* Add support for disabling the on-disk archive for pods with limited disk resources
* Add some cookie tests and fix revealed bugs
* Add start page preference for users (#956)
* Add Health Check to Docker Container (#958)

### Updates

* Update CHANGELOG for 0.15.0
* Update deps and fix timeline and profile view to show no icon for normal users/humans/etc
* Update deps
* Update API docs
* Update go.yarn.social/types
* Update the settings template to improve the experience (#1024)
* Update translactions
* Update deps
* Update known pods


<a name="0.14.0"></a>
## [0.14.0](https://git.mills.io/yarnsocial/yarn/compare/0.13.1...0.14.0) (2022-05-14)

### Updates

* Update CHANGELOG for 0.14.0


<a name="0.13.1"></a>
## [0.13.1](https://git.mills.io/yarnsocial/yarn/compare/0.13.0...0.13.1) (2022-05-14)

### Bug Fixes

* Fix muted message based on UL/OL (#936)
* Fix memory surge by disabling daily stats job (expensive)
* Fix feed page spacing (#927)
* Fix missing lang items
* Fix errors from bad hashes from the /conv handler (#922)
* Fix a panic bug in lookups
* Fix mentions functionality on formatting toolbar (#920)
* Fix multiple image stacking (#918)
* Fix navigation mobile menu with alerts (#910)
* Fix prompt to go back a page on cancel (#908)
* Fix link verification for posts containing internal links (#904)
* Fix embed improt
* Fix postbox keeping text after submission (#900)
* Fix the Dockerfile.dev used by docker-compose.yml
* Fix spacing for single image post for lightbox (#887)
* Fix Profile CSS with no twts (#875)
* Fix missing UnmarshalJSON() for subscriptions
* Fix filtering out subscriptions in UpdateFeeds job
* Fix typo
* Fix bug in Cache that incorrectly creates Twter objects for local feeds to the pod
* Fix WebSub handlers
* Fix link for external muted user (#852)
* Fix structure of HTML for non-inline rendering of media
* Fix image render regression (#835)
* Fix profileLinks partials to not show bookmarks link for external profiles (#832)
* Fix profile view spacing issue (#826)
* Fix toolbar reply postbox alignment (#824)
* Fix invalid import in tests
* Fix Pico CSS style tooltips from being cut off on articles (#818)
* Fix default for DisplayMedia for new pods (Thanks @scara :D)
* Fix User.Fork() behaviour
* Fix install target (#790)
* Fix #783 (#784)

### Features

* Add Release Notes for yarnd v0.14 (#942)
* Add support for displaying Links found in External Feeds in External Profile view (#941)
* Add additional known pods in the wild
* Add Deployment Guide Documentation (#933)
* Add ndl.taigrr.com to known pods
* Add link verification modal (#911)
* Add user preference for stripping third party tracking on posted URLs (#913)
* Add muted message on in-reply-to (#897)
* Add URL verification to all external facing links (#891)
* Add 'readmore' to all h-entry articles (#892)
* Add Read More for all e-content (#888)
* Add json tags
* Add some websub_ metrics (gauges)
* Add a debug /api/v1/websub endpoint for Poderators for easier curl + jq debugging of websub weirdness
* Add a tool for computing the on-disk Archive path for a given Twt Hash (for example to delete it)
* Add topic validation to websub
* Add support for real-time feed fetching using WebSub (#857)
* Add support for User Links in Profile and Feed Metadata (#854)
* Add a more strict bluemonday HTML sanitizer policy (#853)
* Add view for Muted Feeds and Twts/Yarns (#850)
* Add support for muting by Subject and Hash (#849)
* Add netbros.com to known pods
* Add support for IndieAuth endpoint (#799)
* Add network-stats script to measure the size of the network
* Add 2022-02-09-multilineextension.md
* Add yarnd.orbsmart.synology.me to known pods
* Add optional support for setting up a TLS listener with Let's Encrypt support (#785)
* Add alert system for poderators to configure (#788)
* Add twt.rjp.is to known pods
* Add yarnd.orbsmart.synology.me to known list of pods

### Updates

* Update CHANGELOG for 0.13.1
* Update some pod default configuration values
* Update 'preflight.sh' (#899)
* Update deps (CVE-2020-29652)
* Update 'internal/email.go' (#860)
* Update LICENSE to AGPLv3 and update README (#794)
* Update multi-line extension re-@lyse


<a name="0.13.0"></a>
## [0.13.0](https://git.mills.io/yarnsocial/yarn/compare/0.12.0...0.13.0) (2022-02-05)

### Bug Fixes

* Fix style issue for blockquotes with hard line breaks whilst still preserving the fix in #668 (#781)
* Fix RenderImage() bug which was  not permitting title attr on a or img elements during HTML sanitization
* Fix handling of empty lines in feeds (#772)
* Fix Cache.FollowedBy() to return true if user.Is(uri)
* Fix bad Cache.Twters in Cache v20 and reset (letting it rebuild from source data) (#769)
* Fix broken context for blocked feeds config/context
* Fix ParseFile() to return types.ErrInvalidFeed on parser errors resulting in nil lines and only set Cache.Twters on successful parse (#767)
* Fix blocked feeds configuration and add some default blocked feed patterns
* Fix updating Cache.Twters when injecting twts and several other bad bugs found :/ (#760)
* Fix Edit LastTwt so it works properly (#753)
* Fix Privacy / User Preference bug in cache.GetMentions()
* Fix bug in DownloadImage() leaking termporary files for external avatar downloads (#746)
* Fix support for WebMentions (from the web) to /twt/<hash> permalinks as well as /user/<username> links (#741)
* Fix the way we work out what the root twt is in conversation view (#739)
* Fix CI
* Fix WebMentions once and for all (#737)
* Fix bug in deleting user cache views on timeline preference change
* Fix RenderImage() to only append ?full=1 on Media URIs that actually look like they came from a Yarn pod (#735)
* Fix bad event handling when there is no mentioned-list DOM elements :/
* Fix some more JS bugs with the @-mention autocomplete and restore ENTER to complete behaviour (#733)
* Fix ENTER clearing Text in KeyDown event for @-mention autocomplete (use TAB only)
* Fix issue with @ mentions in reply boxes (#732)
* Fix hiding @ mention list on selection (#729)
* Fix minor spelling error (yarml > yaml) (#730)
* Fix broken empty post behaviour (#726)
* Fix removing drawer class on postbox in resetPostBox()
* Fix drawer spacing issue on last yarn (#712)
* Fix %t (text) and %c (compat) for lextwt.Mention node and strip subject hash from Permalink OG and Twitter Card rendering
* Fix tests
* Fix 0f333b3
* Fix a minor bug with IPP and add some lgging
* Fix spacing issues and remove 'register' link in particpate message for registration disabled pods.
* Fix to smaller images scaling to full width
* Fix lack of public followers mapping in FetchFeeds() call in IPP
* Fix FeatureJumpTimelineAge to guard against Divide by Zero (See #688)
* Fix code block
* Fix filter by age parameter that hsould have been a query string
* Fix check for anonymous users (no username)
* Fix Cache.GetByUserView()
* Fix Cache.IsShadowed()
* Fix potential template rendering error when ConversationHanadler() tries to render an empty types.Twts{} slice due to muted feeds

### Features

* Add v0.13 Release Notes (#782)
* Add first e2e integration test for InfoHandler() (#774)
* Add unit tests for invalid feedds (#763)
* Add support for cached pages with reload so Poderators can more easily customize Pod pages (#756)
* Add yarn.yarnpods.com to known pods
* Add support for setting the Reply text to initially @-mentino the Twter() even if the user doesn't follow them (#749)
* Add two additional known pods
* Add Context.View which is the name of the template/view being rendered (#740)
* Add proper validated support for vanity URI(s) e.g: @prologic@twtxt.net (#738)
* Add support for Avatar GIFs :D (#734)
* Add first-class support for GIFs (#727)
* Add support for fetching feeds over gemini://
* Add support for %t (text) rendering to lextwt.Link node to improve OG and Twitter Card rendering
* Add support for Twitter Summary Card (Fixes #706)
* Add FixAdminFeeds job to cleanup @help and @twtxt feeds from AdminUser (Fixes #704)
* Add missing download attribute db39c14 missed
* Add support for pressing Escape to canel a Reply
* Add small highlight to article on post box movement.
* Add support for building and publishing two different Docker images (latest vs dev) (#693)
* Add ability to jump to a certain 'X' hours ago on the timeline (#688)
* Add support for configuring Twts Per Page in Manage Pod

### Updates

* Update CHANGELOG for 0.13.0
* Update langs


<a name="0.12.0"></a>
## [0.12.0](https://git.mills.io/yarnsocial/yarn/compare/0.11.0...0.12.0) (2022-01-11)

### Bug Fixes

* Fix logo alignment. Fix gap issues with code/pre tags. (#680)
* Fix CI
* Fix bad commit in f7db6f9
* Fix StatsJob() and posting stats as the statsBot via the adminUser
* Fix sort order of Hide Replies (I think)
* Fix JSON response of Who Follows Resource (#639)
* Fix language used in ManageJobs() on successful job queue
* Fix the URL for user urls in /whoFollows response
* Fix stupid  bugs caused by c14cebb
* Fix size of icon for profiles with no avatar
* Fix language for ProfileDoesNotFollowYou to be more neutral
* Fix code colouring (#636)
* Fix duplicate followers bug
* Fix bug with updating Follower.LastSeenAt (pointer mishap)

### Features

* Add Release Notes for v0.12.0 (Thanks @Ullarah)
* Add support for Pod-level and User-level Display Media and Original Media settings
* Add support back for original inline image display as a DisplayImagesPreference (inline)
* Add Download link to Image modal dialog to download original quality media
* Add c content-negogiated media view to access original quality media cross-pod (#666)
* Add support for preserving original quality images and serving thumbnail + link to original (#661)
* Add support for configurable pod-level avatar and media resolutions (#663)
* Add support for deleting feed in Manage Pod -> Manage Users
* Add FeatureInternalEvents (internal_events) and basic Onboarding flow for new users
* Add support for Hide Replies user settings (Closes #549)
* Add Avatar to Twter object for User/Feed models so injected Twts injected directly in the cache show the User/Feed's Avatar
* Add temporary backwards compatibility for yarnd 0.11.x for the fucked up /whoFollows response in MultiUserAgent().Follows() logic
* Add a yarnc hash command for constructing Twt Hashes by hand (useful for tests and older/legacy clients)
* Add support for actually nuking the cache with the Refresh Cache admin UI with a Javascript confirmation
* Add Release Notes for v0.11.0

### Updates

* Update CHANGELOG for 0.12.0
* Update Pod Logo and Favicon to simpler colors (#658)
* Update translations
* Update 'AUTHORS' file (#634)


<a name="0.11.0"></a>
## [0.11.0](https://git.mills.io/yarnsocial/yarn/compare/0.10.0...0.11.0) (2022-01-01)

### Bug Fixes

* Fix -R/--raw output of yarnc timeline command
* Fix NaN bug
* Fix logging for FeatureMovingAverageFeedRefresh
* Fix nil pointer bug in Cache.TwtCount()
* Fix leaking temporary files
* Fix leaking temporary files and remove when done
* Fix ShouldRefreshFeed() to always refresh feeds on the same pod
* Fix boundedMovingAverage expression
* Fix minor bug with non-existent/blank refresh interval
* Fix compat for old Twters in archive
* Fix LookupHandler() and associated Javascript to handle External Avatars and only pull up what the user follows
* Fix data race in BaseTask
* Fix an obscure bug with injected twts from peering pods (do not fuck with the LastModified time of the feed)
* Fix User.Unfollow()
* Fix Unfollow behaviour
* Fix TwtxtHandler to display and use correct profile attribute for no. of followers/followings
* Fix source of occassional 500 errors on /conv/:hash links caused by -TwtRootLinkTitle missing from other translations
* Fix permalink template and view to handle muted feeds
* Fix bug displaying Followers count in Settings
* Fix some gaping security holes by adding missing MustAuth() middleware on protected resources requiring auth (#615)
* Fix an IndexError bug in SyndicationHandler
* Fix followers template
* Fix APIv1 compat by partially reverting fb43196
* Fix possible deadlocks in store
* Fix #602: Referrer not remembered after login (#614)

### Features

* Add monthly active users metric (mau)
* Add -p/--page support to yarnc timeline command
* Add support for -n/--twts and --r/--reverse to yarnc timeline command
* Add special case for alwaysRefreshDomains
* Add FeatureMovingAverageFeedRefresh `moving_average_feed_refresh` feature
* Add support for feed refresh intervals + spec + refactor to support handling bad feeds (#624)
* Add ActiveUsers to startup jobs so dau isn't 0 on startup when it's not
* Add ActiveUsers job
* Add PruneFollowers job
* Add the builtin theme to the Docker image at /theme
* Add concurrency to check_pod_versions.go to make it go faster :D
* Add a better tool (written in Go) for checking pod versions

### Updates

* Update CHANGELOG for 0.11.0
* Update .gitignore


<a name="0.10.0"></a>
## [0.10.0](https://git.mills.io/yarnsocial/yarn/compare/0.9.0...0.10.0) (2021-12-20)

### Bug Fixes

* Fix rotate on submit button (#612)
* Fix Dockerfile
* Fix deadlock proper in Converge() and use getPeers() to avoid recursive locks :/
* Fix deadlock between cache.Converge() and cache.GetPeers()
* Fix case where there are no suitable peers from the twts and ask a random subset (60%) of peers for the missing root twt
* Fix search by tags because we normalize the case of tags when indexing in the cache
* Fix posting performance by moving the Cache Convergence to the UpdateFeeds job
* Fix noise from DetectPodFromResponse() for Twtxt responses that are non-yarnd pods with an empty/non-existent Powered-By header
* Fix Follow/Unfollow AJAX
* Fix error message in BaseFromURL()
* Fix settings template
* Fix bad merge
* Fix bad typo
* Fix Cache.Store() and Cache.Load() to take advantage of gob streams and reduce memory allocations on startup
* Fix peer caching for stale peers with lastUpdated > podInfoUpdateTTL
* Fix a bunch issues found withchanges in the UI/UX mostly to do with Follow/Unfollow and refactor profile views
* Fix a bunch of missing UI elements on profile view

### Features

* Add gen-secrets.sh script to provide a convenience way to generate required pod secrets
* Add LastSeen and LastPosted support
* Add metrics for convergence processing time and missing twts
* Add support for cache congerence (missing root twts) (#610)
* Add support for detecting outgoing peers
* Add link to the Twter.URL's baseURL(or Yarn.social pod) (#607)
* Add a Manage Jobs Poderator interface for manually triggering jobs and display background job status
* Add docs for FromOldCache()
* Add support for loading up a previous cache version (n-1) and migrating to current version
* Add MemoryUsage() to help optimize some parts of the codebase's memory allocations

### Updates

* Update CHANGELOG for 0.10.0
* Update 'README.md'
* Update logo
* Update managePeers template to display no. of peers at the top header
* Update Manage Jobs icon


<a name="0.9.0"></a>
## [0.9.0](https://git.mills.io/yarnsocial/yarn/compare/0.8.0...0.9.0) (2021-12-06)

### Bug Fixes

* Fix memory leak of the internal taskMap for async tasks
* Fix cache tests
* Fix display of peers to have a consistent sort order
* Fix tools/inject_twt.sh
* Fix bug causing inconsisstencyin order of (#hash) @mention in forks (left over UniqStringsfrom previous refactor :O)
* Fix form used for Following new feeds (now under Feeds view)
* Fix profile and externalProfile to only show Follow/Unfollow and Mute/Unmute for authenticated users
* Fix media button formatting (#569)
* Fix the location of the tabler-icons.woff icon font and where it's served from
* Fix bug / bad data in customTimeMagnitudes table causing incorrect relative times to be displayed for >1m

### Features

* Add tools/compare_twt_chains.sh for comparing two pods conversation chains for set difference of twts
* Add content-negogiated application/json output for /conv views
* Add quick script for injecting twts from a source pod to a target pod given a hash
* Add support for discovering peering pods (#582)
* Add an experimental injection endpoint for injecting twts into the cache and archive

### Updates

* Update CHANGELOG for 0.9.0
* Update External Avatar for injected Twts
* Update to tabler-icons v1.45.0
* Update 'internal/theme/static/css/99-yarn.css'
* Update 'internal/langs/active.en.toml' (#571)
* Update icons for default theme to use Tabler Icons (#562)


<a name="0.8.0"></a>
## [0.8.0](https://git.mills.io/yarnsocial/yarn/compare/0.7.4...0.8.0) (2021-11-21)

### Bug Fixes

* Fix CHANGELOG template (too noisey)
* Fix loading pod settings with invalid features (Fixes #556)
* Fix IsAdmin contexxt variable
* Fix a potential panic in NewContext()
* Fix bug with --disable-ffmpeg that also disable incorrectly uploading of photos/pictures
* Fix fd socket leak in webmention handling
* Fix the margin of the Yarn count badge
* Fix a few bugs with twt context rendering (aparently you can't nest a tags :O)
* Fix twt context to use proper Markdown rendering + HTML sanitizer (#532)
* Fix css for show_twt_context so ellipsis works
* Fix display of Root conv button for unauthenticated users

### Features

* Add graceful AJAX support for Follow/UnFollow in Following and Followers views (Fixes #547)
* Add support for Service Workers to be hosted at /js/ (/static/js/ in themes)
* Add MagicLinkAuth, StripTwtSubjectHashes and ShowTwtContext as promoted features
* Add graceful AJAX for Follow/UnFollow and Mute/UnMute for externalProfile template too
* Add a bunch of missed defer f.Close() -- none of which I believe could lea file descriptors though from open files (#531)
* Add a class to the twt navigation (#528)
* Add getRootTwt template function
* Add experimental Twt Context feature enabled with show_twt_context
* Add support for displaying list of features with ./yarnd --enable-feature list

### Updates

* Update CHANGELOG for 0.8.0
* Update CHANGELOG for 0.8.0
* Update CHANGELOG for 0.8.0
* Update CHANGELOG template and config
* Update 'internal/theme/static/css/99-yarn.css' (#534)
* Update 'internal/theme/static/css/99-yarn.css' (#533)
* Update default pod logo as per abb9c01


<a name="0.7.4"></a>
## [0.7.4](https://git.mills.io/yarnsocial/yarn/compare/0.7.3...0.7.4) (2021-11-15)

### Bug Fixes

* Fix dupe Twt bug in User views by returning a copy of the view's slice
* Fix broken logic for archived root twts

### Updates

* Update CHANGELOG for 0.7.4


<a name="0.7.3"></a>
## [0.7.3](https://git.mills.io/yarnsocial/yarn/compare/0.7.2...0.7.3) (2021-11-15)

### Bug Fixes

* Fix cache bug causing dupe Root Twts in conversation views (Caching is hard 😅)
* Fix conversation forking
* Fix twt-hash link to lower-right of Twt cards to always be linked
* Fix bug disabling features in the Manage Pod UI
* Fix CI
* Fix UI/UX behaviour of the Bookmark button

### Features

* Add experimental StripConvSubjectHashes (strip_conv_subject_hashes) feature
* Add (re-add) MergeStore daily job
* Add conversation length badges and fix Edit/Delete/Reply/Conv buttons (can't use ids here)

### Updates

* Update CHANGELOG for 0.7.3
* Update 'internal/theme/templates/settings.html' (#521)
* Update tools/check-versions.sh with new pod: txt.quisquiliae.com


<a name="0.7.2"></a>
## [0.7.2](https://git.mills.io/yarnsocial/yarn/compare/0.7.1...0.7.2) (2021-11-13)

### Bug Fixes

* Fix Docker image to just work without any arguments and drop chown of default data volume

### Updates

* Update CHANGELOG for 0.7.2


<a name="0.7.1"></a>
## [0.7.1](https://git.mills.io/yarnsocial/yarn/compare/0.7.0...0.7.1) (2021-11-13)

### Bug Fixes

* Fix signal handling using su-exec and ownership of default data volume /data
* Fix example Docker Compose for Docker Swarm to match Docker image entrypoint chagne
* Fix password reset token by properly checking counters (#519)
* Fix password reset token by not decrementing it twice in the token cache
* Fix expiryAt timestamp for magic_link auth and password reset tokens
* Fix example Docker Compose for Docker Swarm

### Features

* Add an improved Docker image that supports PUID/PGID env vars to run yarnd as different users (e.g: to support Synolgy)

### Updates

* Update CHANGELOG for 0.7.1


<a name="0.7.0"></a>
## [0.7.0](https://git.mills.io/yarnsocial/yarn/compare/0.6.2...0.7.0) (2021-11-13)

### Bug Fixes

* Fix bug in setting tokens for password reset and magiclink auth
* Fix minor bug in TTLCache.get()
* Fix cached tokens to be deleted after use
* Fix token expiry check on password reset form
* Fix password reset and magiclink auth tokens to only be usable once with a default TTL of 30m
* Fix edit/reply/fork buttons ot use ID selectors
* Fix /custom static route for prod builds
* Fix incorrect cache header used in conversation view
* Fix FeatureFlags' JSON marshalling/unmarshalling
* Fix Cached.UpdateFeed() to not overwrite a cached feed with an empty set of twts
* Fix misspelled translation key for ErrTokenExpired
* Fix autocomplete for Login via EMail's username field
* Fix potential nil pointer bug in PermalinkHandler()
* Fix bug that logs pod owner out when deleting users
* Fix the session cookie name once and for all 🥳
* Fix Pod Base URL automatically if it's missing a Scheme, log a warning and assume https://
* Fix caching bug with editing and deleting last twt
* Fix default MediaResolution to match default themes page width of 850px (Fixes #508)
* Fix Cache.FetchTwts() to Normalize Feed.URL(s) before doing anything
* Fix Session Cookie name to by config.LocalURL().Hostname() so the Pod name can be more free form
* Fix NormalizeURL() to strip URL Fragments from Feed URIs
* Fix short time ago formats (Fixes #509)
* Fix title entries of Atom/XML feeds

### Features

* Add improved UX for Follow/Unfollow and Mute/Unmute with graceful JS fallback
* Add support for Pod-level (User overridable) Timezone, Time and external link preferences
* Add support for /custom/*filepath arbitrary static file serving from a theme with any directory/file structure
* Add missing lang msg for MsgMagicLinkAuthEmailSent
* Add support for Login via Email with feature magic_link_auth
* Add isFeatureEnabled() func to templates
* Add support for custom pages (Fixes #393)
* Add s simple JS confirm to logout to avoid accidental logout
* Add a PruneUsers job that runs once per week with an email list of candidate users to delete for the Pod Owner based on some heuristics
* Add new pod yarn.meff.me to manually track versions to help keep the ecosystem up-to-date as we grow :D
* Add basic validation of the Pod's Base URL to ensure it is TLS enabled in non-debug (production) mode
* Add pagination extension spec (#494)

### Updates

* Update CHANGELOG for 0.7.0
* Update deps
* Update contributing guidelines


<a name="0.6.2"></a>
## [0.6.2](https://git.mills.io/yarnsocial/yarn/compare/0.6.1...0.6.2) (2021-11-08)

### Bug Fixes

* Fix consistency of pointers on toolbar buttons (Fixes #505)
* Fix expression in User Settings template for which openLinksInPreference is checked 🤦‍♂️
* Fix typo in User Settings handler for persisting OpenLinksInPreference 🤦‍♂️

### Features

* Add RotateFeeds job (#504)
* Add article target (#506)
* Add per-handler latency metrics

### Updates

* Update CHANGELOG for 0.6.2


<a name="0.6.1"></a>
## [0.6.1](https://git.mills.io/yarnsocial/yarn/compare/0.6.0...0.6.1) (2021-11-06)

### Bug Fixes

* Fix a nil pointer exception handling User OpenLinksInPreference :/

### Updates

* Update CHANGELOG for 0.6.1


<a name="0.6.0"></a>
## [0.6.0](https://git.mills.io/yarnsocial/yarn/compare/0.5.0...0.6.0) (2021-11-06)

### Bug Fixes

* Fix and improve the yarnc timeline to support Markdown rendering and fix multi-line twts
* Fix missing parser.HardLineBreak extension
* Fix consistency of Pod URL for the yarnc command-line client to match the mobile app just taking a Pod URL
* Fix broken tests (#500)
* Fix typo in recovery email template
* Fix bug persisting BlacklistedFeeds in Pod override Settings (ooops)
* Fix validation/initialization bugs with pod override settings and handling of whitelisted_images and blacklisted_feeds
* Fix data races with feature flags
* Fix marshaling/unmarshaling of feature flags
* Fix some weird Markdown parsing and HTML formatting behaviour by turning off some extensions/options
* Fix bug with serializing FeatureFlags causing pod settings overrides not to load
* Fix typo (case) in managePod template
* Fix a panic bug in Cache v7 :/
* Fix preflight check for Go1.17
* Fix performance of computing timeline view updated_at timestamps
* Fix updating UpdateAt timestamps across all views
* Fix the builtin theme to use title instead of data-tooltip
* Fix Markdown html rendering and re-enable CommonFlags
* Fix performance regression posting as user that e9222f5 broke
* Fix metadata parsing in comments (#484)
* Fix TwtxtHanadler() to support If-Modified-Since and Range requests (#490)

### Features

* Add a Open links in to user settings page
* Add a user setting OpenLinksInPreference to control behaviour of opening external links
* Add highlights to mentions for the yarnc command-line client's timeline sub-command
* Add vim support to yarnc post to use an editor to write the twt
* Add parser.NoEmptyLineBeforeBlock to Markdown parsing extensions
* Add support for user preferred time format (12h/24h)
* Add support for blacklisting feeds or entire pods, domains, etc
* Add background tasks for Post handlers and External feeds to decrease latency
* Add custom user views that take into account user muted feeds
* Add a --disable-logger flag for those that already do access logs at the ingress level
* Add a local cache view
* Add a new view for Discover
* Add a hacky pre-rendering hook and move timeline tiemstamp updates there for hopefully better perf
* Add a --max-cache-fetchers configuration option to control the number of fetchers the feed cache uses (defaults to number of cpus)
* Add a custom version of humanize.Time with shorter output
* Add .Profile.LastPostedAt to be used by template authors
* Add support for displaying last updated times for the main three timeline views and proivde as template context (updated_at)
* Add support for displaying last updated times for the main three timeline views and proivde as template context
* Add support for displaying external feed's following feeds
* Add immediate on-pod feed/persona updates
* Add retryableStore to automatically recover from database failure in some cases (#474)
* Add discover_all_posts and shorter_permalink_title as permanent feature changes
* Add fork button to search results
* Add a much impmroved SearchHandler() and new search template with similar UX to Conversation/Yarn view (#493)

### Updates

* Update CHANGELOG for 0.6.0


<a name="0.5.0"></a>
## [0.5.0](https://git.mills.io/yarnsocial/yarn/compare/0.4.1...0.5.0) (2021-10-29)

### Bug Fixes

* Fix unnecessary warning on non-existent settings.yaml on startup (Fixes #435)
* Fix italics formatting toolbar butotn to use _foo_ instead of *foo*
* Fix the name of the default database, app specific JS and CSS files
* Fix loading custom templates and static assets with -t/--theme
* Fix mising AvatarHash for the API SettingsEndpoint()
* Fix missing translation on Conversation View (Yarn) title

### Features

* Add Feeds to Profile for the Mobile App (only for the user viewing their own profile)
* Add support for ignoring twts in the future until they become relevant (#491)
* Add tw.lohn.in to tools/check-versions.sh
* Add automatic rename of old store path for pod owners that use defaults to smooth over change in 034009e
* Add first-class support for themes to more easily customize templates and static assets
* Add support for displaying bookmarks in a timeline view instead of just a list of hashes
* Add two additional lines of preamble text promoting Yarn.social (shameful plug)
* Add support for also disabling media in general (--disable-media)
* Add -m 5 (5s timeout) to tools/check-pod-versions.sh

### Updates

* Update CHANGELOG for 0.5.0


<a name="0.4.1"></a>
## [0.4.1](https://git.mills.io/yarnsocial/yarn/compare/0.4.0...0.4.1) (2021-10-25)

### Updates

* Update CHANGELOG for 0.4.1
* Update the default favicon
* Update release script to also push the git commit


<a name="0.4.0"></a>
## [0.4.0](https://git.mills.io/yarnsocial/yarn/compare/0.3.0...0.4.0) (2021-10-24)

### Bug Fixes

* Fix abbr of URL
* Fix padding on code blocks (#476)

### Features

* Add support for auto-refreshing avatars using a cache-busting technique
* Add a fragment, a blake2b base32 encoded hash of user and feed avatars to avatar metddata
* Add list of available features if invalid feature supplied to --enable-feature

### Updates

* Update CHANGELOG for 0.4.0


<a name="0.3.0"></a>
## [0.3.0](https://git.mills.io/yarnsocial/yarn/compare/0.2.0...0.3.0) (2021-10-24)

### Bug Fixes

* Fix nav layout and remove unncessary container-fluid class
* Fix excessive padding, minor CSS tweaks and use a centered layout (#473)
* Fix bug with User.FollowAndValidate() to use heuristics to set an alias for the feed if none given
* Fix typo in ad-hoc check-versions.sh tool
* Fix performance regression of loading timeline
* Fix CI
* Fix the Web interface UI/UX to be a bit more compact and cleaner (#468)
* Fix CI
* Fix badge color to match primary color
* Fix video poster generation for short (<3s) videos using thumbnail filter
* Fix User.Fork() behaviour
* Fix an edge case with the lextwt feed parser with bad url/avatar metadata missing the scheme
* Fix bug where external avatars were not generated for some misbehaving feeds
* Fix the rendering of the Twter in permalink titles
* Fix overriding alrady cached/discovered Avatars from feed advertised Avatar in feed preamble metadata
* Fix Reply hints to just the Twter
* Fix and remove unnecessary footer on every Markdown page
* Fix tests
* Fix external feed avatar handling so avatars are always cached and served from the pod
* Fix up the rendering of the default pages
* Fix the yarnc stats sub-command to display the right information on feeds
* Fix User.Filter() to return an empty slice if all twts are filtered
* Fix handling for external avatars so the local pod has a cached copy and services all external avatars (regression)
* Fix external profile view to correctly point to the external profile view for the header feed  name and cleanup some of hte code
* Fix consistency of profileLinks partial and drop use of TwtURL from Profile model
* Fix internal links to open in same page

### Documentation

* Document todo for removing fragment on mentions from the API as Goryon may not need it now

### Features

* Add arrakis.netbros.com to ad-hoc tools/check-versions.sh
* Add Follow to Settings page and fix template reloading in debug mode (#454)
* Add conversation length badges
* Add shell script to check versions of known pods
* Add support for templating the logo with the pod name and update the default yarnd logo
* Add test case for weird bug with adi's twt (#463)

### Updates

* Update CHANGELOG for 0.3.0
* Update 'internal/static/css/01-pico.css'
* Update 'internal/static/css/01-pico.css'
* Update 'internal/static/css/01-pico.css'
* Update 'internal/static/css/99-twtxt.css'
* Update 'internal/static/css/01-pico.css'
* Update 'internal/langs/active.en.toml'
* Update 'internal/templates/base.html'
* Update 'internal/static/css/01-pico.css'
* Update 'internal/static/css/99-twtxt.css'
* Update 'internal/static/css/99-twtxt.css'
* Update deps
* Update some of the default options
* Update 'internal/templates/page.html'
* Update 'internal/langs/active.en.toml'


<a name="0.2.0"></a>
## [0.2.0](https://git.mills.io/yarnsocial/yarn/compare/0.1.0...0.2.0) (2021-10-17)

### Bug Fixes

* Fix GoReleaser config
* Fix bug in profileLinks partial
* Fix external profile view to not display following/followers counts if zero
* Fix NFollowing count based on no. of # follow = found
* Fix Conversation view to hide Root button when the conversation has no more roots
* Fix display of Fork button
* Fix typo
* Fix layout of local feeds
* Fix Feeds view when uneven no. of feeds
* Fix Feeds view
* Fix behaviour of when Fork/Conversation buttons are displayed contextually and cleanup UX
* Fix behaviour of prompt for when the Timeline is empty
* Fix Python twt hash reference implementation (#459)
* Fix serving videos behind Cloudflare for Safari and Mobile Safari (redo of 35241bc) by adding a --disable-gzip flag
* Fix broken test
* Fix mentions lookup and expansion based on the new (by default) nick@domain format with fallback to local user and custom aliases
* Fix shortel_permalink_title maxPermalinkTitle to 144
* Fix feature for shorter_permalink_title
* Fix serving videos for Mobile Safari / Safari through Cloudflare by disabling Gzip compression :/
* Fix related projects
* Fix up the README's links and branding
* Fix follow handlers and data models to cope with following multiple feeds with the same nick or alias
* Fix typo in Conversations sub-heading view
* Fix showing Register button when registrations are disabled (#452)
* Fix inconsistent button colors using only contrast class sparingly and only for irréversible actions like deleting your account (#453)
* Fix tests
* Fix duplicate twtst in FilterOutFeedsAndBotsFactory
* Fix missing view context initialization missed in two other templates
* Fix templtes view context var that wasn't initialized
* Fix logic of handling local user/feeds vs. external feeds in /api/v1/fetch-twts endpoint
* Fix fetching twts from external profiles
* Fix order of twts in conversations (reverse order)
* Fix Twts.Swap() implementation
* Fix sorting of twts to be consistent (sorted by created, then hash)
* Fix typo in error handling for updating white list domains
* Fix preflight to handle Go 1.17
* Fix loading settings and re-applying WhitelistedDomains on startup
* Fix lextwt to cope with bad url in metadata
* Fix FeedLookup case sensitivity bug
* Fix CI
* Fix preflight to account for null GOPATH and GOBIN
* Fix performance of processing WebMentions by using a TaskFunc
* Fix Makefile
* Fix typo in +RegisterFormEmailSummary
* Fix error handling in reset user
* Fix image name for Docker Hub
* Fix Drone CI config to build docker images in paralell and fix installation of deps
* Fix Drone CI for building and pushing Docker images using plugins/kaniko plugin
* Fix writeBtn (blog post editor) behavior (#433)
* Fix version string of server binary
* Fix Drone CI config and add make deps back
* Fix Drone CI and Docker image tags and versioning
* Fix Drone CI config
* Fix server version strings
* Fix typo
* Fix Dockerfile.dev with missing langs soruces
* Fix some more dependneices of things that moved to git.mills.io
* Fix docker-compose
* Fix .gitignore
* Fix stale issue workflow
* Fix stale issues workflow
* Fix change in case of translation files
* Fix User-Agent detection by tightening up the regex used (#418)
* Fix lang translations to use embed and io/fs to load translations
* Fix whitespace
* Fix display of multi-line private messages
* Fix Emails removeal migration and remove it from the templates
* Fix tests and refactor invalid feed error (#412)
* Fix bug in URLForConvFactory to also check for OP Hash validity in the archive too
* Fix bug in lastTwt.Created() call -- missed in refactor from concrete types to interfaces
* Fix formatTwtText template factory function
* Fix User-Agent matching for iPhone OS 14.x that can't handle WebP correctly (even though it claims it can)
* Fix typo around custom pod Settings struct with missing quote for pod_logo
* Fix custom pod logo support (#358)
* Fix public following disclosure (#379)
* Fix an off-by-one error in types.SplitTwts() (#377)
* Fix Uhuntu build (#376)
* Fix option for -P/--parser
* Fix fallback to config theme when user has no theme set
* Fix display of published time of blog posts (#366)
* Fix User Profile URL routes (#360)
* Fix typo
* Fix typo
* Fix typos (#353)
* Fix a bug for iOS 14.3 Mobile Safari not rendering WebP correctly (work-around)
* Fix missing closing a (Fixes #335)
* Fix working simplified docker-compose.yml to get up and running quickly
* Fix a JS bug clikcing on the Write button in the toolbar to start writing a Twt Blog Post
* Fix internal events from non-Public IPs (#332)
* Fix Referer redirect behaviour (#331)
* Fix deleting last twt (missing csrf token)
* Fix goreleaser configs
* Fix bug in UnparseTwtFactory() causing erroneous d to be appended to domains
* Fix footer layout regression (#322)
* Fix messaging for when feeds may possibly exceed conf.MaxFetchLimit
* Fix the way cache_limited counter/log works
* Fix CI
* Fix HTML validation errors (#321)
* Fix text overflow p element
* Fix a JS bug on load
* Fix tag expansion (#314)
* Fix user able visit login and register page after login (#313)
* Fix edge case with Subject parsing
* Fix developer experience so UI/UX developers can modify templates and static assets without reloading or rebuilding
* Fix header wrapping on domain (#306)
* Fix action URL for managing pods
* Fix Docker image build
* Fix Docker image push
* Fix Docker Image badges (don't yet have a jointwt Docke rHub org)
* Fix Docker image push
* Fix conversations for Twt Blog Posts (#287)
* Fix blog post validation
* Fix bad data with missing Followers
* Fix spelling errors on register form
* Fix bitcask dep
* Fix a data race with session handling
* Fix refreshing user's own feed and re-warming cache on a user deleting their last Twt (#272)
* Fix blog posting with empty titles (#271)
* Fix image Makefile target that 15f25c1 broke and add PUBLISH=1 variable so we publish new images again
* Fix order of twts displayed in twt timeline (reverse order)
* Fix data privacy by removing all user email addresses and never storing emails (#269)
* Fix image target to not push the image built immediately (CI does this)
* Fix LICENSE file
* Fix image resizing for media (avatars was already working)
* Fix bug with AddFollower()
* Fix timeline style on mobile devices with very long feed names (#248)
* Fix userAgentRegex bug with greedy matching
* Fix UA detection and relax regex even more
* Fix logic in archiver makePath
* Fix bug in archiver to prevent use of invalid twt hashes
* Fix concurrent map read/write bug with feed cache
* Fix Follows/FollowedBy/Muted for external profiles
* Fix followers view for external profiles
* Fix bad copy/paste
* Fix Docker build
* Fix title of twt permalinks
* Fix generate Make target to run rice to embed the assets __after__ minification/bundling
* Fix size of RSS Icon on External Profile view
* Fix avatar for external profiles when no avatar found
* Fix size of icss-rss icon for externals feeds without an avatar
* Fix color of multimedia upload buttons
* Fix bug in /settings view
* Fix more UI/UX things on the Web App with better CSS
* Fix typo
* Fix spacing around icons for Post/Publish button (#233)
* Fix potential session cookie bug where Path was not explicitly set
* Fix name of details settings tab for Pod Management
* Fix name of details settings tab for Pod Management
* Fix /report view to work anonymously without a user account
* Fix conversation bug caused by user filter (mute user feature)
* Fix a bug with User.Filter()
* Fix wording in Mute / Report User text
* Fix grammar on /register view
* Fix link to /abuse page
* Fix JS error
* Fix og:description meta tag
* Fix a few bugs with OpenGraph tag generation
* Fix URLForExternalProfile links and typo (#224)
* Fix subject bug causing conversations to fork (#217)
* Fix JS bug with persisting title/text fields
* Fix API Signing Key (#212)
* Fix video upload quality by disabling rescaling (#205)
* Fix structtag for config (#203)
* Fix old missing twts missing from the archive from local users and feeds
* Fix older Twt Blog posts whoose Twts has been archived (and missing? bug?)
* Fix MediaHandler to return a Media Not Found for non-existent media rather than a 500 Internal Server Error
* Fix build (#198)
* Fix responsive video size on mobile (#195)
* Fix video aspect ratio and scaling (#191)
* Fix Range Requests for /media/* (#190)
* Fix video display to use block style inside paragraphs (#189)
* Fix missing names to feeds
* Fix links to external feeds
* Fix ExpandTags() function so links to Github issue comments work and add unit tess (#176)
* Fix PublishBlogHandler(o and fix line endings so rendering happens correctly
* Fix incorrect locking used in GetByPrefix()
* Fix blogs cache bug
* Fix logic for when to attempt to send webmentions for mentions
* Fix duplicate @mentions in reply (Fixes #167)
* Fix concurrency bug in cache.GetByPrefix()
* Fix content-type and set charset=utf-8
* Fix wrapping behaviour and remove warp=hard
* Fix blogs cache to be concurrenct safe too
* Fix global feed cache concurrency bug
* Fix all invalid uses of footer inside hgroup and make all hgroups consistently use h2/h3
* Fix regex patterns for valid username and feed names and mention syntax
* Fix and restore full subjet in twt replies (#163)
* Fix the pager (properly)
* Fix bug with pager (temporary fix)
* Fix nil pointer in map assignment bug
* Fix cli build nad refactor username/password prompt handling
* Fix the CSS/JS bundled assets with new minify tool (with bug fixes)
* Fix bug in /settings with incorrect call to .Profile()
* Fix date/time shown on blog posts (remove time)
* Fix metric naming consistency feed_sources -> cache_sources (#155)
* Fix duplicate tags and mentions (#154)
* Fix content-negotiation for image/webp
* Fix ExpandMentions()
* Fix dns issues in container and force Go to use cgo resolver
* Fix CI
* Fix feed cache bug not storing Last-Modified and thereby not respecting caching
* Fix inconsistency in Syndication and Profile views accessing feed fiels directly instead of by the cache
* Fix CI
* Fix Dockerfile with missing webmention sources
* Fix webmention handling to be more robust and error proof
* Fix AvatarHandler that was incorrectly encoding the auto-generated as image/webp when image/png was asked for
* Fix bug in /mentions and dedupe twts displayed
* Fix bug causing permalink(s) on fresh new twts to 404 when linked to but not in the local pod's cache
* Fix /mentions view logic (ooops)
* Fix bug in register user on fresh pods with no directory structure in place yet
* Fix typo
* Fix bug
* Fix ordering of twts post cache ttl and archival
* Fix an index out of bounds bugs
* Fix the logic for Max Cache Items (whoops)
* Fix bug in ParseFile() which _may_ cause all local and user twts to be considered old
* Fix UX perf of posting and perform async webmentions instead
* Fix bugs with followers/folowing view post external feed integration
* Fix session leadkage by not calling SetSession() on newly created sessions (only when we actually store session data)
* Fix leaking sessions and clean them up on startup and only persist sessions to store for logged in users with an account
* Fix external avatar, don't set a Twter.Avatar is none was foudn
* Fix build
* Fix bug with caching Twter.URL for external feeds
* Fix isLocal check in proile template
* Fix more bus
* Fix Twtter caching in feed cache cycles
* Fix Twtxt URL of externalProfile view
* Fix several bugs to do with external profiles (I'm tired :/)
* Fix typo
* Fix bug with DownloadImage()
* Fix computed external avatar URIs
* Fix external avatar negogiation
* Fix old user avatars (proper)
* Fix profile avatars
* Fix profile bug
* Fix Dockerfile (again)
* Fix Dockerfile adding missing minify tool
* Fix Edit on Github page link
* Fix bug in #hashtag searching showing duplicates twts in results
* Fix the global feed_cache_size count
* Fix profile view and make Reply action available on profile views too
* Fix typo
* Fix bug in theme so null theme defaults to auto
* Fix feed_cache_size metric bug (missed counting already-cached items)
* Fix bug in profile view re ShowConfig for profileLinks
* Fix another memory optimization and remove another cache.GetAll() call
* Fix media upload image resize bug
* Fix caching bug with /avatar endpoint
* Fix similar bug to #124 for editing last twt
* Fix bug in feed cache for local twts
* Fix bug in feed cache dealing with empty feeds or null urls
* Fix media uploads to only resize images if image > UploadOptions.ResizeW
* Fix session storage leakage and delete/expunge old sessions
* Fix memory allocation bloat on retrieving local twts from cache
* Fix the glitchy theme-switcher that causes unwanted flicker effects (by removing it!)
* Fix populating textarea when reply button is clicked for more than one time. (#124)
* Fix repository name in Docker GHA workflow
* Fix database fragmentation and merge on startup
* Fix local twts lookup  to be O(n) where n is the no. of source urls (not total twts in cache)
* Fix image handling and auto-orient images according to their EXIF orientation tag
* Fix panic in PageHandler() -- not safe to reuse the markdown parser, etc

### Documentation

* Document Twt Subject Extension (#309)

### Features

* Add support for detecting and disabling ffmpeg support for audio/video with --disable-ffmpeg as an optional configuration flag
* Add support for fetching, storing and serving remote external feed description and following/followers counts via the API
* Add support for fetching, storing and displaying remote feed description and following/followers counts
* Add Root to link back to root conversations in Conversation view
* Add Disable Gzip server setting on startup output
* Add build-dev-site Drone CI job to build docs (dev.twtxt.net) site
* Add Spec for Metadata Extension (#451)
* Add shorter_permalink_title feature
* Add yarnc --help to README
* Add the same nick@domain and then nick_xx behaviour for keeping track of followers too
* Add Tools section to Useer Settings and Bookmarklet that can be added to browser bookmark bars
* Add some prompts to new users with an empty timeline
* Add a human/crawler friendly version of /version
* Add a SoftwareConfig struct and /version endpoint to more easily identify pods
* Add JSON marshallers/unmarshallers to FeatureFlags
* Add FeatureDiscoverALlPosts to the API end /api/v1/discover endpoint too
* Add optional feature (--enable-feature discover_posts_all)
* Add following and followers counts to default feed preamble metadata
* Add a hacky temporary solution to squelch flip-flopping followers (#450)
* Add a ~/:nick/post/ route for blog posts
* Add support for Muting/Unmuting external pods (cross-pod)
* Add a POSIX Shell script to run preflight checks as part of the default Makefile target
* Add handlers for ~/:nick
* Add --json flag to yarnc timeline command for seeing and processing the raw JSON from the API
* Add WhitelistedDomains support to Manage Pod settings
* Add content-negogiation for /twt/:hash handler
* Add CI for dev.twtxt.net
* Add native support for Gopher (gopher://) feeds
* Add support for forking conversations in the Web UI
* Add Drone CI step to build and push to prologic/yarnd (Docker Hub)
* Add a 2nd docker image publihs step
* Add separate docker image publish step
* Add Reset User feature to Pod Management interface
* Add a blake2b_base32 cli tool
* Add i18n to deps and generate active.*.toml
* Add missing lang (i18n) files to Docker image builds
* Add preamble to top of all twtxt feeds (#384)
* Add check for Bad Request in CLI and fix PEBKAC problem of entering a bad Pod API Base URL :D
* Add support for bookmarks (#372)
* Add support for draft blog posts and fix deletion (#367)
* Add Node.js reference implementation of Twt Hash algorithm (#365)
* Add @twtxt FOLLOW events for feeds too
* Add information about ffmpeg dependency. Remove repeated FreeBSD section. Fix a formatting error or two
* Add missing CI step for tools deps
* Add local Drone CI config
* Add support for expanding Twtxt mentions and tags in Twt Blog posts
* Add support for bookmarklet(s)
* Add Content-Length to TwtxtHandler for /user/:user/twtxt.txt URIs so Pods know what the size of feeds are
* Add feed_limited counter to measure no. of feeds affected by conf.MaxFetchLimit
* Add SameSite session cookie policy and CSRF Verification to prevent Cross-Site-Sciprint (XSS)
* Add msgs.Inc() and msgs.Dec() for messages
* Add Pod config handlers and API endpoints (#297)
* Add rice-embed.go so the go get works
* Add net/http/pprof to debug mode
* Add improved async media upload endpoint for the API (backwards compatible) (#274)
* Add front matter to pages
* Add Atom link for Pod's timeline (aka Discover) on the bottom of every page
* Add ValidateFeeds job to cleanup bad feeds on twtxt.net
* Add timeline command to twt (command-line client)
* Add GET support for /api/v1/settings endpoint to retrieve user settings/object (#256)
* Add /api/v1/settings API Endpoint for updating user settings (#252)
* Add better support for UA on fetches with token callback (#249)
* Add Follows/FollowedBy attributes to user profiles
* Add footnote parsing for Twt Blogs
* Add HardLineBreak and NoEmptyLineBeforeBlock to Markdown parser options
* Add protection against brute force login attempts at user accounts (#241)
* Add FilterTwts to conversation view and treat /conv URIs a bit like /twt (permalink) ones
* Add a RealIP middleware to capture the real ip of clients when we're behind a proxy
* Add better page titles to improve SEO
* Add /api/v1/conv ConversationEndpoint() to API
* Add DEBUG=1 capability to Makefile for debug builds with real-time static (css/js/img) asset modifications
* Add missing FilterTwts() calls to API
* Add Muted property to Profile objects in ProfileResposne for the API (#235)
* Add improved CSS for the timeline view
* Add feed validation to Follow requests from Web App and API to avoid invalid feeds (#228)
* Add /mute and /unmute API endpoints (#230)
* Add /support and /report API endpoints (#231)
* Add switch on /register view to encourage new users to read the community guidelines and agree to them (EULA)
* Add support for Pod Owners to manage users (add/delete) (#227)
* Add a fast-path to User.Filter() when .muted is of length 0
* Add support for muting/unmuting intolerable users (#226)
* Add text to /register view that links to the /abuse page
* Add blank Abuse Policy (to be filled out)
* Add OpenGraph Meta tags to Twt Permalinks (#225)
* Add support for uploading audio media (#222)
* Add MarkdownText field to Twt (#223)
* Add Tags and Subject as dynamic fields to Twt.MarshalJSON() for API clients (#219)
* Add support for managing a pod's configuration as a pod owner (#170)
* Add Hash to Twt and Slug to Twter outputs when Marshalled as JSON for the API (#206)
* Add archive_dupe counter
* Add support for persisting Title and Text in Twts and Twt Blogs to local storage (#193)
* Add support for auto-generating poster thumbnails for uploaded videos
* Add playsinline as allowed attr on video elements
* Add ffmeg to Dockerfile runtime image  and remove debug loggging (#188)
* Add support for older video media (locally) (#187)
* Add support for video uploads and hosting (#186)
* Add blogs, archived and cache stats and cache_blogs to metrics
* Add BlogPost PublishedAt and Modified timestamps
* Add support for editing twt blog posts (#171)
* Add missing mu.Lock() for loading older twts cache
* Add support for newlines in Twts (short-form) by using Unicode LS (\u2028) to encode them (#166)
* Add Timeline navbar item and cleanup Navbar
* Add support for editing/deleting your last Twt in a Conversation view
* Add Ctrl+Enter to Post a new Twt
* Add link to Blog on Twts associated with a Twt Blog Post
* Add formatDateTime template function to shorten the human display of Twt posted date/time shorter when it wasn't that long ago
* Add a border around avatar images
* Add support for style attr on some HTML elements as an option for some users
* Add integrated support for conversations (#161)
* Add a O(1) lookup for a types.Twt by hash in the global feed cache (cahced on demand)
* Add /blogs view to display all twt blog posts for an author (#160)
* Add unit tests for types.Twt.Subject() and fix the regex (#157)
* Add unit tests for types.Feed (#156)
* Add a note about having to be logged in to comment on blog posts
* Add better UI/UX around the blog view with comments in reverse order and some nice headings
* Add support for long-form posts (blog posts) with integrated twt(s). (#152)
* Add remote mention syntax @user@domain (#151)
* Add twtd_server_sessions metric
* Add Token model and view for managing API Tokens (#135)
* Add vendored version of github.com/prologic/webmention since it has detracted slightly from standard webmention handling
* Add a Remember Me to /login view
* Add a new SessionStore that is a caching/persistent store
* Add MergeStoreJob to merge the store once per day
* Add local twts to mentions view in addition to feeds the user follows
* Add a DiskArchver that implements a Cache TTL (#138)
* Add more candidates to GetExxternalAvatar()  and rearrange preferences
* Add a db.Merge() call after startupJobs() to free memory on database compaction possibly after some cleanup jobs
* Add a GenerateAvatar() func and use github.com/nullrocks/identicon for identicons
* Add loading=lazy attr to all images to lazy load images on supported browsers
* Add post form to external profile view
* Add improved external feed integration
* Add .gitkeep files for data directory layout
* Add missing HEAD handler for /externa/:slug/avatar
* Add support for WebP image format (#130)
* Add tooling to combined and minify all static css/js assets into a single file and request (boosting load times)
* Add external avatar to /external view
* Add support for fetching, caching and displaying external avatars
* Add /robots.txt view
* Add post form on profile view and puts profile links in a 2nd column
* Add user preferences for theme in /settings view
* Add target=_blank to Config/Twtxt/Atom Profile Links
* Add link to user config on /settings view
* Add server_info metric with version information
* Add Grafana Dashbaord
* Add /user/:nick/config.yaml handler (Closes #36)
* Add debug-only memory profiler to debug some memory bloat
* Add a Uploading to the Upload button tooltip when its processing the media
* Add a paper-plance icon to the Post button
* Add profile links to /settings view to be consistent with the profile view

### Updates

* Update CHANGELOG for 0.2.0
* Update the footer of the base template to remove James Mills (the original author/creator) and instead link to Yarn.social as the primary branding
* Update to Bitcask v0.3.14
* Update bitcask to v0.3.13
* Update README
* Update README a little bit and add Drone CI badge
* Update docker image publication
* Update deps
* Update about.md
* Update stale workflow
* Update privacy.md
* Update icon used for bookmarking Twts (Closes Viglino/iconicss#21)
* Update production example docker + traefik deployment
* Update README.md
* Update README.md
* Update deps
* Update README
* Update external feed source URI(s)
* Update Go module path to github.com/jointwt/twtxt and remove committed .min.* and commit rice-embed.go insted
* Update links to Related Projects
* Update README.md
* Update bundled CSS
* Update abuse.md
* Update about.md
* Update Grafana Dashboard
* Update Grafana Dashboard
* Update Grafana Dashboard
* Update Grafana Dashboard
* Update Grafana Dashboard
* Update Grafana Dashboard
* Update Grafana Dashboard
* Update Grafana Dashboard
* Update Grafana Dashboard
* Update Grafana Dashboard
* Update README.md
* Update AUTHORS (#131)
* Update Grafana Dashboards and make all panels transparent
* Update Grafana Dashbaord with fixed Go Memory panel
* Update Grafana Dashbaord
* Update README.md
* Update README.md


<a name="0.1.0"></a>
## [0.1.0](https://git.mills.io/yarnsocial/yarn/compare/0.0.12...0.1.0) (2020-08-19)

### Bug Fixes

* Fix paging on discover and profile views
* Fix missing links in about page
* Fix Dockerfile with missing new pages
* Fix horizontal scroll / overflow on mobile devices
* Fix Atom feed and populate Summary with text/html and title with text/plain
* Fix UX of hashes and shorten them to 11 (by default) characters which is roughly 88 bits of entropy or basically never likely to collide :D
* Fix UX of relative time display and use humanize.Time
* Fix /settings to be a 2-column layout since we don't have that many settings
* Fix superfluous paragraphs in twt formatting
* Fix the email templates to be consistent
* Fix the UX of the password reset view
* Fix formatting of Support Request email and indent/quote Subject/Message
* Fix the workding around password reset emails
* Fix Reply-To for support emails
* Fix email to send text/plain instead of text/html
* Fix wrong template for SendSupportRequestEmail()
* Fix Docker GHA workflow
* Fix docker image versioning
* Fix Docker image
* Fix long option name for open registrations
* Fix bug in /lookup handler
* Fix /lookup to only regturn following and local feeds
* Fix /lookup handler behaviour
* Fix UI/UX of relative twt post time
* Fix UI/UX of date/time of twts
* Fix Content-Type on HEAD /twt/:hash
* Fix a bunch of IE11+ related JS bugs
* Fix Follow/Unfollow actuions on /following view
* Fix feed_cache_last_processing_time_seconds unit
* Fix bug with /lookup handler and perform case insensitive looksup
* Fix and tidy up the /settings view with followers/following now moved to their own views
* Fix missing space on /followers
* Fix user experience with editing your last Twt and preserve the original timestamp
* Fix Atom URL for individual Twts (Fixes #117)
* Fix bad name of PNG (typod extension)
* Fix hash collisions of twts by including the source twtxt URI as well
* Fix and add some missing icons
* Fix bug in new permalink handling
* Fix other missing uploadoptions

### Features

* Add post partial to permalink view for authenticated users so Reply works
* Add WebMentions and basic IndieWeb µFormats v2 support (h-card, h-entry) (#122)
* Add missing spinner icon
* Add tzdata package to runtime docker image
* Add user setting to display dates/times in timezone of choice
* Add Content-Typre to HEAD /twt/:hash handler
* Add HEAD handler for /twt/:hash handler
* Add link to twt.social in footer
* Add feed_cache_last_processing_time_seconds metric
* Add /metrics endpoint for monitoring
* Add external feed (#118)
* Add link to user's profile from settings
* Add Follow/Unfollow actions for the authenticated user on /followers view
* Add /following view with defaults for new to true and tidy up followers view
* Add Twtxt and Atom links to Profile view
* Add a note about off-Github contributions to README
* Add PNG version of twtxt.net logo
* Add support for configurable img whitelist (#113)
* Add permalink support for individual local/external twts (#112)
* Add etags for default avatar (#111)
* Add text/plain alternate rel link to user profiles
* Add docs for Homebrew formulare

### Updates

* Update CHANGELOG for 0.1.0
* Update CHANGELOG for 0.0.13
* Update README.md
* Update README gif (#121)
* Update /feeds view and simplify the actions and remove own feeds from local feeds as they  apprea in my feeds already
* Update the /feeds view with My Feeds and improve some of the wording
* Update README.md (#116)
* Update README.md
* Update logo
* Update README.md


<a name="0.0.12"></a>
## [0.0.12](https://git.mills.io/yarnsocial/yarn/compare/0.0.11...0.0.12) (2020-08-10)

### Bug Fixes

* Fix duplicate build ids for goreleaser config
* Fix and simplify goreleaser config
* Fix avatar upload handler to resize (disproportionally?) to 60x60
* Fix config file loading for CLI
* Fix install Makefile target
* Fix server Makefile target
* Fix index out of range bug in API for bad clients that don't pass a Token in Headers
* Fix z-index of the top navbar
* Fix logic of count of global followers and following for stats feed bot
* Fix the style of the media upload button and create placeholde rbuttons for other fomratting
* Fix the mediaUpload form entirely by moving it outside the twtForm so it works on IE
* Fix bug pollyfilling the mediaUpload input into the uploadMedia form
* Fix another bug with IE for the uploadMedia capabilities
* Fix script tags inside body
* Fix JS compatibility for Internet Explorer (Fixes #96)
* Fix bad copy/paste in APIv1 spec docs
* Fix error handling for APIv1 /api/v1/follow
* Fix the route for the APIv1 /api/v1/discover endpoint
* Fix error handling of API's isAuthorized() middleware
* Fix updating feed cache on APIv1 /api/v1/post endpoint
* Fix typo in /follow endpoint
* Fix the alignment if the icnos a bit
* Fix bug loading last twt from timeline and discover
* Fix delete last tweet behaviour
* Fix replies on profile views
* Fix techstack document name
* Fix Dockerfile image versioning finally
* Fix wrong handler called for /mentions
* Fix mentions parsing/matching
* Fix binary verisoning
* Fix Dockerfile image and move other sub-packages to the internal namespace too
* Fix typo in profile template

### Documentation

* Document Bitcask's usage in teh Tech Stack

### Features

* Add Homebrew tap to goreleaser config
* Add version string to twtd startup
* Add a basic CLI client with login and post commands (#108)
* Add hashtag search (#104)
* Add FOLLOWERS:%d and FOLLOWING:%d to daily stats feed
* Add section to /help on whot you need to create an account
* Add backend handler /lookup for type-ahead / auot-complete @mention lookups from the UI
* Add tooltip for toolbar buttons
* Add &nbsp; between toolbar sections
* Add strikethrough and fixed-width formatting buttons on the toolabr
* Add other formatting uttons
* Add support for syndication formats (RSS, Atom, JSON Feed) (#95)
* Add Pull Request template
* Add Contributor Code of Conduct
* Add Github Downloads README badge
* Add Docker Hub README badges
* Add docs for the APIv1 /api/v1/post and /api/v1/follow endpoints
* Add configuration open to have open user profiles (default: false)
* Add basic e2e integration test framework (just a simple binary)
* Add some more error logging
* Add support for editing and deleting your last Twt (#88)
* Add Contributing section to README
* Add a CNAME (dev.twtxt.net) for developer docs
* Add some basic developer docs
* Add feature to allow users to manage their subFeeds (#80)
* Add basic mentions view and handler
* Add Docker image CI (#82)
* Add MaxUploadSizd to server startup logs
* Add reuseable template partials so we can reuse the post form, posts and pager

### Updates

* Update CHANGELOG for 0.0.12
* Update CHANGELOG for 0.0.12
* Update CHANGELOG for 0.0.12
* Update CHANGELOG for 0.0.12
* Update /about page
* Update issue templates
* Update README.md
* Update APIv1 spec docs, s/Methods/Method/g as all endpoints accept a single-method, if some accept different methods they will be a different endpoint


<a name="0.0.11"></a>
## [0.0.11](https://git.mills.io/yarnsocial/yarn/compare/0.0.10...0.0.11) (2020-08-02)

### Bug Fixes

* Fix size of external feed icons
* Fix alignment of Twts a bit better (align the actions and Twt post time)
* Fix alignment of uploaded media to be display: block; aligned
* Fix postas functionality post Media Upload (Missing form= attr)
* Fix downscale resolution of media
* Fix bug with appending new media URI to text input
* Fix misuse of pronoun in postas dropdown field
* Fix sourcer links in README
* Fix bad error handling in /settings endpoint for missing avatar_file (Fixes #63)
* Fix potential vulnerability and limit fetches to a configurable limit
* Fix accidental double posting
* Fix /settings handler to limit request body
* Fix followers page (#53)
* Fix wording on settings re showing followers publicly
* Fix bug that incorrectly redirects to the / when you're posting from /discover
* Fix profile template and profile type to show followers correctly with correct link
* Fix Profile.Type setting when calling .Profile() on models
* Fix a few misisng trimSuffix calls in some tempaltes
* Fix sessino persistence and increase default session timeout to 10days (#49)
* Fix session unmarshalling caused by 150690c
* Fix the mess that is User/Feed URL vs. TwtURL (#47)
* Fix user registration to disallow existing users and feeds
* Fix the specialUsernames feeds for the adminuser properly on twtxt.net
* Fix remainder of feeds on twtxt.net (we lost the contents of news oh well)
* Fix new feed entities on twtxt.net
* Fix all logging in background jobs  to only output warnings
* Fix and tidy up dependencies

### Features

* Add /api/v1/follow endpoint
* Add /api/v1/discover endpoint
* Add /api/v1/timeline endpoint and content negogiation for general NotFound handler
* Add a basic APIv1 set of endpoints (#66)
* Add Media Upload Support (#69)
* Add Etag in AvatarHandler (#67)
* Add meta tags to base template
* Add improved mobile friendly top navbar
* Add logging for SMTP configuration on startup
* Add configuration options for SMTP From addresss used
* Add fixPossibleFeedFollowers migration for twtxt.net
* Add avatar upload to /settings (#61)
* Add update email to /settings (Fixees #55
* Add Password Reset feature (#51)
* Add list of local (sub)Feeds to the /feeds view for better discovery of user created feeds
* Add Feed model with feed profiles
* Add link to followers
* Add random tweet prompts for a nice variance on the text placeholder
* Add user Avatars to the User Profile view as well
* Add Identicons and support for Profile Avatars (#43)
* Add a flag that allows users to set if the public can see who follows them

### Updates

* Update CHANGELOG for 0.0.11
* Update README.md
* Update README
* Update and improve handling to include conventional (re ...) (#68)
* Update pager wording
* Update pager wording  (It's Twts)
* Update CHANGELOG for 0.0.11
* Update default list of external feeds and add we-are-twtxt
* Update feed sources, refactor and improve the UI/UX by displaying feed sources by source instead of lumped together


<a name="0.0.10"></a>
## [0.0.10](https://git.mills.io/yarnsocial/yarn/compare/0.0.9...0.0.10) (2020-07-28)

### Bug Fixes

* Fix bug in ExpandMentions
* Fix incorrect log call
* Fix server shutdown and signal handling to listen for SIGTERM and SIGINT
* Fix twtxt.net missing user feeds for prologic (home_datacenter) wtf?!
* Fix missing db.SetUser for fixUserURLs
* Fix another bug in Profile template
* Fix more bugs with User Profile view
* Fix User Profile Latest Tweets
* Fix build and remove unused vars in FixUserAccounts
* Fix User URL and TwtURLs on twtxt.net  and reset them
* Fix Context.IsLocal bug
* Fix bug in User.Is function
* Fix /settings view (again)
* Fix build error (oops silly me)
* Fix bug in /settings vieew
* Fix missing feeds for @rob and @kt84  that went missing from their accounts :/
* Fix UI/UX bug in text input with erroneous spaces
* Fix adminUser account on twtxt.net
* Fix user feeds on twtxt.net
* Fix bug with feed creation (case sensitivity)
* Fix Follow/Unfollow local events post v0.9.0 release re URL/TwtURL changes
* Fix numerous bugs post v0.8.0 release (sorry) due to complications  with User Profile URL vs. Feed URL (TwtURL)
* Fix Tweeter.URL on /discover
* Fix syntax error (oops)
* Fix adminUser feeds on twtxt.net
* Fix link to user profiles in user settings followers/following
* Fix Tagline in User Settings so you users can see what they have entered (if it was set)
* Fix User.Following URIs post v0.9.0 break in URIs

### Features

* Add fixAdminUser function to FixUserAccountsJob to add specialUser feeds to the configured AdminUser
* Add SyncStore job to sync data to disk every 1m to prevent accidental data loss
* Add logging when server is shutdown and store is synced/closed
* Add local @mention automatic linking for local users and local feeds without a user having to follow  them first

### Updates

* Update CHANGELOG for 0.0.10
* Update README.md
* Update README.md
* Update README.md
* Update startup to merge data store
* Update deps
* Update the FixUserAccounts job and remove all fixes, but leave  the job (we might breka more things)
* Update FixUserAccounts job and remov most of the migration code now that twtxt.net data is fixed
* Update FixUserAccounts job schedule to @hourly and remove adminUser.Feeds hack
* Update  FixUserAccountsJob to eif User URL(s)
* Update FixUserAccounts job back to 1h schedule


<a name="0.0.9"></a>
## [0.0.9](https://git.mills.io/yarnsocial/yarn/compare/0.0.8...0.0.9) (2020-07-26)

### Features

* Add user profile pages and **BREAKS** existing local user feed URIs (#27)

### Updates

* Update CHANGELOG for 0.0.9


<a name="0.0.8"></a>
## [0.0.8](https://git.mills.io/yarnsocial/yarn/compare/0.0.7...0.0.8) (2020-07-26)

### Bug Fixes

* Fix the custom release-notes for goreleaser (finally)
* Fix the gorelease custom release notes by skipping the gorelease changelog generation
* Fix the release process, remove git-chglog use before running gorelease
* Fix instructions on how to build from source (Fixes #30)
* Fix PR blocks and remove autoAssign workflow that fails with Resource not accessible by integration
* Fix releasee process to generate release-notes via git-chglog which are better than goreleaser's ones
* Fix goarch in gorelease config (uggh)
* Fix gorelease config (3rd time's the charm)
* Fix gorelease config properly (this time)
* Fix release tools and remove unused shell script
* Fix goreleaser config
* Fix binary release tools and process
* Fix name of twtxt Docker Swarm Stackfile
* Fix docker stack usage in README (Fixes #34)
* Fix typo in feeds template
* Fix error handling for user registrationg and return 404 Feed Not Found for non-existent users/feeds
* Fix build error (oops)
* Fix set of reserved vs. special usernames
* Fix unconstrained no. of user feeds to prevent abuse
* Fix error message when trying to register an account with a previously deleted user (to preserve feeds)
* Fix potential abuse of unconstrained username lengths
* Fix and remove  some useless debugging
* Fix documentation on configuration options and warn about user registration being disabled (Fixes #29)
* Fix the annoying greeting workflow and remove it (it's kind of annoying)

### Features

* Add default configuration env values to docker-compose (#39)
* Add git-chglog to release process to update the ongoing CHANGELOG too
* Add feed creation event to the twtxt special feed
* Add FixUserAccounts job logic to touch feed files for users that have never posted
* Add automated internal special feed
* Add documentation on using docker-compose (#26)
* Add new feature for creating sub-feeds / personas allowing users to create topic-based feeds and poast as those topics
* Add a section to the help page on formatting posts

### Updates

* Update CHANGELOG for 0.0.8
* Update CHANGELOG for 0.0.8
* Update CHANGELOG for 0.0.8
* Update CHANGELOG for 0.0.8
* Update CHANGELOG for 0.0.8
* Update CHANGELOG for 0.0.8
* Update CHANGELOG for 0.0.8


<a name="0.0.7"></a>
## [0.0.7](https://git.mills.io/yarnsocial/yarn/compare/0.0.6...0.0.7) (2020-07-25)

### Bug Fixes

* Fix .gitignore for ./data/sources
* Fix bug updating followee Followers
* Fix poor spacing between posts on larger devices (Fixes #28)
* Fix and remove accidently commited data/sources file (data is meant to be empty)
* Fix bug with Follow/Unfollow and updating Followers, missed using NormalizeUsername()
* Fix updating Followers on Follow/Unfollow for local users too
* Fix potential nil map bug
* Fix user accounts and populate local Followers
* Fix the settings Followers Follow/Unfollow state
* Fix build system to permit passing VERSION and COMMIT via --build-arg for docker build
* Fix the CI builds to actually build the daemon (#21)

### Features

* Add a convenient UI/UX [Reply] feature on posts
* Add twtxt special feed updates for Follow/Unfollow events from the local instance
* Add confirmation on account deletion in case of accidental clicks
* Add support for faster Docker builds by refactoring the Dockerfile (#20)
* Add Docker Swarmmode Stackfile for production deployments based on https://twtxt.net/ (#22)
* Add local (non-production) docker-compose.yml for reference and Docker-based development (#25)

### Updates

* Update NewFixUserAccountsJob to 1h schedule


<a name="0.0.6"></a>
## [0.0.6](https://git.mills.io/yarnsocial/yarn/compare/0.0.5...0.0.6) (2020-07-23)

### Bug Fixes

* Fix formatting in FormatRequest
* Fix the session timeout (which was never set0
* Fix some embarassing typos :)
* Fix error handling for parsing feeds and feed sources

### Features

* Add bad feed dtection and log possible bad feeds (no action taken yet)
* Add new feature to detect new followers of feeds on twtxt.net from twtxt User-Agent strings
* Add twtxt to reserve usernames
* Add an improved /about page and add a /help page to help new users

### Updates

* Update README and remove references to the non-existent CLI (this will come later)
* Update default job interval of UpdateFeedSourcesJob


<a name="0.0.5"></a>
## [0.0.5](https://git.mills.io/yarnsocial/yarn/compare/0.0.4...0.0.5) (2020-07-21)

### Bug Fixes

* Fix UI/UX handling around bad logins
* Fix the follow self feature properly with more consistency
* Fix firefox UI/UX issue by upgrading to PicoCSS v1.0.3 (#17)

### Features

* Add /feeds view with configurable feed sources for discoverability of other sources of feeds
* Add username validation to prevent more potential bad data
* Add configurable theme (site-wide) and persist user-defined (vai cookies) theme selection (#16)
* Add configurable maximum tweet length and cleanup tweets before they are stored to replace new lines, etc


<a name="0.0.4"></a>
## [0.0.4](https://git.mills.io/yarnsocial/yarn/compare/0.0.3...0.0.4) (2020-07-21)

### Bug Fixes

* Fix links opening in new window with target=_blank
* Fix typo on support page (#5)
* Fix app versioning and add to base template so we can tell which version of twtxt is running
* Fix bug in TwtfileHandler with case sensitivity of nick param

### Features

* Add delete account support
* Add better layout of tweets so they stand out more
* Add support for Markdown formatting (#10)
* Add pagination support (#9)
* Add Follow/Unfollow to /discover view that understands the state of who a user follows or doesn't (#8)

### Updates

* Update README.md
* Update README.md


<a name="0.0.3"></a>
## [0.0.3](https://git.mills.io/yarnsocial/yarn/compare/0.0.2...0.0.3) (2020-07-19)

### Bug Fixes

* Fix bug with NormalizeURL() incorrectly translating https:// to http://
* Fix deps and cleanup unused ones
* Fix the layout of thee /settings view
* Fix a critical bug whereby users could re-register the same username and override someone else's account :/
* Fix username case sensitivity and normalize by forcing all usersnames to be lowercase and whitespace stripped
* Fix useability issue where some UI/UX would toggle between authenticated and unauthentiated state causing confusion

### Features

* Add support for configuring flags from the environment via the same long option names
* Add options to configure session cookie secret and expiry
* Add Contributing guideline and TODO
* Add additional logic to fix null/blank user account URL(s) to the FixUserAccountJob as well
* Add FixUserAccountsJob to fix existing broken accounts that might already exist
* Add new /discover view for convenience access to the global instance's timeline  with easy to use Follow links for discoverability


<a name="0.0.2"></a>
## [0.0.2](https://git.mills.io/yarnsocial/yarn/compare/0.0.1...0.0.2) (2020-07-19)

### Bug Fixes

* Fix the  follow self behaviour to actually work
* Fix defaults to use the same  ones in twtxt's options
* Fix  URL() of User objects
* Fix import and hard-code no. of tweets to display

### Features

* Add feature whereby new registered users follow themselves by default
* Add support, stargazers and contributing info to READmE
* Add enhanced server capability for graceful/clean shutdowns
* Add /import feature to import multiple feeds at once (#1)

### Updates

* Update feed update frequency to 5m by default


<a name="0.0.1"></a>
## 0.0.1 (2020-07-18)

### Bug Fixes

* Fix release tool
* Fix screenshots
* Fix broken links and incorrect text that hasn't happened yet
* Fix /login CTA link on /register page
* Fix /register CTA link on /login page
* Fix parsing store uri
* Fix bug ensuring feedsDir exists
* Fix Dockerfile

### Features

* Add theme-switcher and remove erroneous prism.js

### Updates

* Update README.md

