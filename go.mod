module git.mills.io/yarnsocial/yarn

go 1.24.0

// Fixes a regression in github.com/gomarkdown/markdown that is causing images
// to be rendered with junk "/> after the closing image tag :/ But also re-introduces
// the Markdown parsing bug that @bender found in https://github.com/gomarkdown/markdown/issues/309
replace github.com/gomarkdown/markdown => github.com/gomarkdown/markdown v0.0.0-20221013030248-663e2500819c

require (
	git.mills.io/prologic/go-gopher v0.0.0-20220331140345-72e36e5710a1
	git.mills.io/prologic/observe v0.0.0-20210712230028-fc31c7aa2bd1
	git.mills.io/prologic/read-file-last-line v0.0.0-20210710073401-af293d63a6d0
	git.mills.io/prologic/useragent v0.0.0-20210714100044-d249fe7921a0
	github.com/KimMachineGun/automemlimit v0.7.1
	github.com/Masterminds/sprig/v3 v3.3.0
	github.com/NYTimes/gziphandler v1.1.1
	github.com/PuerkitoBio/goquery v1.10.3
	github.com/andreadipersio/securecookie v0.0.0-20131119095127-e3c3b33544ec
	github.com/andyleap/microformats v0.0.0-20150523144534-25ae286f528b
	github.com/angelofallars/htmx-go v0.5.0
	github.com/audiolion/ipip v1.0.0
	github.com/badgerodon/ioutil v0.0.0-20150716134133-06e58e34b867
	github.com/creasty/defaults v1.8.0
	github.com/cyphar/filepath-securejoin v0.4.1
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/disintegration/gift v1.2.1
	github.com/disintegration/imageorient v0.0.0-20180920195336-8147d86e83ec
	github.com/dustin/go-humanize v1.0.1
	github.com/gabstv/merger v1.0.1
	github.com/gavv/httpexpect/v2 v2.16.0
	github.com/glebarez/sqlite v1.11.0
	github.com/go-mail/mail v2.3.1+incompatible
	github.com/goccy/go-yaml v1.17.1
	github.com/gomarkdown/markdown v0.0.0-20250311123330-531bef5e742b
	github.com/goware/urlx v0.3.2
	github.com/h2non/filetype v1.1.3
	github.com/hashicorp/golang-lru v1.0.2
	github.com/james4k/fmatter v0.0.0-20150827042251-377c8ea6259d
	github.com/julienschmidt/httprouter v1.3.0
	github.com/justinas/nosurf v1.2.0
	github.com/makeworld-the-better-one/go-gemini v0.13.1
	github.com/marksalpeter/token/v2 v2.0.0
	github.com/mgutz/ansi v0.0.0-20200706080929-d51e80ef957d
	github.com/microcosm-cc/bluemonday v1.0.27
	github.com/mitchellh/go-homedir v1.1.0
	github.com/naoina/toml v0.1.1
	github.com/nicksnyder/go-i18n/v2 v2.6.0
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/rickb777/accept v0.0.0-20170318132422-d5183c44530d
	github.com/robfig/cron v1.2.0
	github.com/robfig/cron/v3 v3.0.1
	github.com/rrivera/identicon v0.0.0-20240116195454-d5ba35832c0d
	github.com/russross/blackfriday v1.6.0
	github.com/sasha-s/go-deadlock v0.3.5
	github.com/securisec/go-keywords v0.0.0-20200619134240-769e7273f2ed
	github.com/simukti/sqldb-logger v0.0.0-20230108155151-646c1a075551
	github.com/simukti/sqldb-logger/logadapter/logrusadapter v0.0.0-20230108155151-646c1a075551
	github.com/sirupsen/logrus v1.9.3
	github.com/slok/go-http-metrics v0.13.0
	github.com/spf13/cobra v1.9.1
	github.com/spf13/pflag v1.0.6
	github.com/spf13/viper v1.20.1
	github.com/srwiley/oksvg v0.0.0-20221011165216-be6e8873101c
	github.com/srwiley/rasterx v0.0.0-20220730225603-2ab79fcdd4ef
	github.com/steambap/captcha v1.4.1
	github.com/stretchr/testify v1.10.0
	github.com/theplant-retired/timezones v0.0.0-20150304063004-f9bd3c0ef9db
	github.com/tj/go-editor v1.0.0
	github.com/unrolled/logger v0.0.0-20201216141554-31a3694fe979
	github.com/wblakecaldwell/profiler v0.0.0-20150908040756-6111ef1313a1
	go.mills.io/bitcask/v2 v2.1.3
	go.mills.io/sessions v0.0.0-20230102023727-1d4fd809624f
	go.mills.io/tasks v0.0.0-20250126215735-c2a885cf816a
	go.mills.io/webfinger v0.0.0-20230218075238-e709ef684f28
	go.mills.io/webmention v0.0.0-20230423111544-baccd6f1042b
	go.sour.is/passwd v0.2.0
	go.yarn.social/client v0.0.0-20250420114029-410ad71a453e
	go.yarn.social/lextwt v0.1.9
	go.yarn.social/types v0.0.0-20250421063104-18007f2aace2
	golang.org/x/crypto v0.37.0
	golang.org/x/net v0.39.0
	golang.org/x/sync v0.13.0
	golang.org/x/term v0.36.0
	golang.org/x/text v0.24.0
	golang.org/x/time v0.6.0
	gopkg.in/yaml.v2 v2.4.0
	willnorris.com/go/microformats v1.2.0
)

require (
	dario.cat/mergo v1.0.1 // indirect
	github.com/TylerBrock/colorjson v0.0.0-20200706003622-8a50f05110d2 // indirect
	github.com/ant0ine/go-webfinger v0.0.0-20150209052316-f8a1773b0e03 // indirect
	github.com/apex/log v1.9.0 // indirect
	github.com/blang/semver/v4 v4.0.0 // indirect
	github.com/captncraig/cors v0.0.0-20190703115713-e80254a89df1 // indirect
	github.com/fatih/color v1.15.0 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/hashicorp/go-immutable-radix/v2 v2.1.0 // indirect
	github.com/hashicorp/golang-lru/v2 v2.0.7 // indirect
	github.com/mattetti/filebuffer v1.0.1 // indirect
	github.com/mitchellh/go-wordwrap v1.0.1 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/pbnjay/memory v0.0.0-20210728143218-7b4eea64cf58 // indirect
	github.com/sagikazarmark/locafero v0.9.0 // indirect
	github.com/sanity-io/litter v1.5.5 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	moul.io/http2curl/v2 v2.3.0 // indirect
)

require (
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/semver/v3 v3.3.1 // indirect
	github.com/PuerkitoBio/purell v1.2.1 // indirect
	github.com/abcum/lcp v0.0.0-20201209214815-7a3f3840be81 // indirect
	github.com/ajg/form v1.5.1 // indirect
	github.com/andybalholm/brotli v1.1.0 // indirect
	github.com/andybalholm/cascadia v1.3.3 // indirect
	github.com/aymerick/douceur v0.2.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/fatih/structs v1.1.0 // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/glebarez/go-sqlite v1.21.2 // indirect
	github.com/go-viper/mapstructure/v2 v2.2.1 // indirect
	github.com/gofrs/flock v0.12.1 // indirect
	github.com/golang/freetype v0.0.0-20170609003504-e2365dfdc4a0 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/gorilla/css v1.0.1 // indirect
	github.com/gorilla/websocket v1.5.1 // indirect
	github.com/grokify/html-strip-tags-go v0.1.0 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/huandu/xstrings v1.5.0 // indirect
	github.com/imkira/go-interpol v1.1.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/marksalpeter/sugar v0.0.0-20160713164314-a69afe358ea8 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/naoina/go-stringutil v0.1.0 // indirect
	github.com/nxadm/tail v1.4.11 // indirect
	github.com/onsi/ginkgo v1.16.5 // indirect
	github.com/onsi/gomega v1.27.10 // indirect
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	github.com/petermattis/goid v0.0.0-20250319124200-ccd6737f222a // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/client_golang v1.22.0 // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/common v0.63.0 // indirect
	github.com/prometheus/procfs v0.16.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/sergi/go-diff v1.3.1 // indirect
	github.com/shopspring/decimal v1.4.0 // indirect
	github.com/spf13/afero v1.14.0 // indirect
	github.com/spf13/cast v1.7.1 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasthttp v1.55.0 // indirect
	github.com/writeas/go-strip-markdown/v2 v2.1.1 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	github.com/xeipuuv/gojsonschema v1.2.0 // indirect
	github.com/yalp/jsonpath v0.0.0-20180802001716-5cc68e5049a0 // indirect
	github.com/yudai/gojsondiff v1.0.0 // indirect
	github.com/yudai/golcs v0.0.0-20170316035057-ecda9a501e82 // indirect
	golang.org/x/exp v0.0.0-20250408133849-7e4ce0ab07d0 // indirect
	golang.org/x/image v0.26.0 // indirect
	golang.org/x/sys v0.37.0 // indirect
	google.golang.org/protobuf v1.36.6 // indirect
	gopkg.in/alexcesaro/quotedprintable.v3 v3.0.0-20150716171945-2caba252f4dc // indirect
	gopkg.in/mail.v2 v2.3.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	gorm.io/gorm v1.25.7 // indirect
	modernc.org/libc v1.22.5 // indirect
	modernc.org/mathutil v1.5.0 // indirect
	modernc.org/memory v1.5.0 // indirect
	modernc.org/sqlite v1.23.1 // indirect
)
