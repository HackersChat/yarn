package internal

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gavv/httpexpect/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/html"
)

// This file contains a patched httpexpect version that adds more convenience
// methods for cookies and HTML forms. Not all functions are overridden, only
// those, that are needed in the tests. So if you encounter a missing method,
// just add and implement it.

func e(t *testing.T) *Expect {
	return &Expect{httpexpect.WithConfig(httpexpect.Config{
		BaseURL: makeURL(),
		Client: &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		Reporter: httpexpect.NewAssertReporter(t),
		Printers: []httpexpect.Printer{
			httpexpect.NewCompactPrinter(t),
			//		httpexpect.NewDebugPrinter(t, true),
		},
	}), t}
}

type Expect struct {
	expect *httpexpect.Expect
	t      *testing.T
}

func (e *Expect) GET(path string, pathargs ...interface{}) *Request {
	return &Request{e.expect.GET(path, pathargs...), e.t}
}

func (e *Expect) POST(path string, pathargs ...interface{}) *Request {
	return &Request{e.expect.POST(path, pathargs...), e.t}
}

type Request struct {
	*httpexpect.Request
	t *testing.T
}

func (r *Request) WithResponseCookie(responseCookie *httpexpect.Cookie) *Request {
	cookie := responseCookie.Raw()
	require.NotNil(r.t, cookie, "cookie cannot be nil")
	r.Request.WithCookie(cookie.Name, cookie.Value)
	return r
}

func (r *Request) WithForm(form interface{}) *Request {
	r.Request.WithForm(form)
	return r
}

func (r *Request) Expect() *Response {
	return &Response{r.Request.Expect(), r.t}
}

type Response struct {
	*httpexpect.Response
	t *testing.T
}

func (r *Response) Status(status int) *Response {
	r.Response.Status(status)
	return r
}

// NoCookie verifies that the response does not have a Set-Cookie header with
// the given cookie name.
func (r *Response) NoCookie(name string) *Response {
	cookieNames := r.Cookies().Iter()
	assert.NotContainsf(r.t, cookieNames, name, "expected response without cookie '%s', but got such a cookie", name)
	return r
}

// ClearCookie verifies that the reponse does have a Set-Cookie header with the
// given cookie name, an empty signed value, max-age of 0 and expiration
// timestamp of 0001-01-01T00:00:00Z.
func (r *Response) ClearCookie(name string) *Response {
	cookie := r.Cookie(name)
	cookie.Value().Match(`^|\d+|[a-zA-Z0-9_-]+$`) // pipe-separated empty value, unix timestamp and signature
	cookie.MaxAge().Equal(0 * time.Second)
	cookie.Expires().Equal(time.Date(1, time.January, 1, 0, 0, 0, 0, time.UTC))
	return r
}

// HTMLForm parses an HTML form from the response body. If parsing fails, the
// test is aborted.
//
// Watch out, not all controls are supported, only <input> tags that specify
// both name and value attributes.
//
// The implementation is also very, very crude. Multiple forms would be merged
// together into a larger form and later fields override earlier fields. This
// can all be fixed once needed.
func (r *Response) HTMLForm() *HTMLForm {
	doc, err := html.Parse(strings.NewReader(r.Body().Raw()))
	require.NoError(r.t, err, "parsing response body HTML failed")

	form := make(map[string]string)
	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		// Currently, we only need input fields. Thus, textareas, buttons etc.
		// are not supported at the moment. Also, both name and value must be
		// specified. So basically only pre-filled fields are parsed.
		if n.Type == html.ElementNode && n.Data == "input" {
			var name, value string
			var foundName, foundValue bool
			for _, attr := range n.Attr {
				switch attr.Key {
				case "name":
					name, foundName = attr.Val, true
				case "value":
					value, foundValue = attr.Val, true
				}
			}
			if foundName && foundValue {
				form[name] = value
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}
	traverse(doc)

	return &HTMLForm{form, r.t}
}

type HTMLForm struct {
	form map[string]string
	t    *testing.T
}

// Field verifies that a HTML form field the with given name exists and returns
// its value.
func (f *HTMLForm) Field(name string) string {
	value, ok := f.form[name]
	if !ok {
		if len(f.form) == 0 {
			f.t.Errorf("expected HTML form field '%s' in response body, but got no supported HTML form fields", name)
		} else {
			var fields []string
			for k, v := range f.form {
				fields = append(fields, fmt.Sprintf("'%s' = '%s'", k, v))
			}
			f.t.Errorf("expected HTML form field '%s' in response body,\n"+
				"but only got %d supported HTML form fields:\n%s",
				name, len(f.form), strings.Join(fields, "\n"))
		}
	}
	return value
}

// NoField verifies that there is no HTML form field with the given name.
func (f *HTMLForm) NoField(name string) *HTMLForm {
	if value, ok := f.form[name]; ok {
		f.t.Errorf("expected no HTML form field '%s' in reponse body, but got one with value '%s'", name, value)
	}
	return f
}
