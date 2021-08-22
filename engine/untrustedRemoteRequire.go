package engine

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"

	"github.com/dop251/goja"
)

type (
	untrustedRemoteRequire struct {
		root       *rootRequire
		origin     *url.URL
		baseURL    *url.URL
		httpClient *http.Client
	}
)

func newRemote(root *rootRequire, target string) (*untrustedRemoteRequire, error) {
	u, err := url.Parse(target)
	if err != nil {
		return nil, err
	}
	// TODO: check if the scheme is http or https, if not fail!
	if u.Scheme == "" {
		return nil, errors.New("remote modules must use HTTP or HTTP/S")
	}
	ur := &untrustedRemoteRequire{
		// TODO: create a separated http client for remote downloads
		httpClient: http.DefaultClient,
		origin:     computeOrigin(u),
		root:       root,
	}
	return ur.sub(u), nil
}

func (r *untrustedRemoteRequire) require(name string) goja.Value {
	if r.root.isBuiltin(name) {
		return r.root.requireFromRemote(name)
	}
	// TODO: currently, it is impossible to use builtin modules from untrusted sources
	// relax this restriction so `some` builtin modules can be loaded.
	target, err := url.Parse(name)
	if err != nil {
		panic(r.root.e.runtime.NewGoError(fmt.Errorf("module %v cannot be parsed as a valid module path", name)))
	}
	if target.Scheme != "" {
		// TODO: check if the scheme is http or https, if not fail!
		// treat it as absolute URL
		if !r.sameOrigin(target) {
			return (r.newOrigin(target)).require(name)
		}
	} else {
		// target is a relative path
		target = r.absURL(target.Path)
	}

	// TODO: remove the number of calls to target.String()
	if module := r.root.hasModule(target.String()); module != nil {
		return module.exports
	}
	r.root.saveModule(target.String(), r.loadModule(name, target))
	return r.root.hasModule(target.String()).exports
}

func (r *untrustedRemoteRequire) javascriptRequire(call goja.FunctionCall) goja.Value {
	name := call.Argument(0).ToString().Export().(string)
	return r.require(name)
}

func (r *untrustedRemoteRequire) loadModule(name string, target *url.URL) *moduleDef {
	code, err := r.parseCode(name, target)
	if err != nil {
		panic(r.root.e.runtime.NewGoError(fmt.Errorf("Unable to parse %v, cause: %w", name, err)))
	}
	moduleOutput, err := r.root.e.runtime.RunProgram(code)
	if err != nil {
		panic(r.root.e.runtime.NewGoError(fmt.Errorf("Unable to load %v, cause %w", name, err)))
	}
	loader, ok := goja.AssertFunction(moduleOutput)
	if !ok {
		panic("This should never ever happen! There something really really wrong with jtb!!!")
	}
	this := r.root.e.runtime.NewObject()
	exports := r.root.e.runtime.NewObject()

	sub := r.sub(target)

	requireFn := r.root.e.runtime.ToValue(sub.javascriptRequire)

	this.Set("require", requireFn)
	this.Set("exports", exports)
	_, err = loader(this, exports, requireFn)
	if err != nil {
		// err is a GoError
		panic(err)
	}
	return &moduleDef{
		exports: exports,
	}
}

func (r *untrustedRemoteRequire) parseCode(name string, url *url.URL) (*goja.Program, error) {
	bytes, err := r.downloadCode(url)
	if err != nil {
		return nil, err
	}

	safeCode := fmt.Sprintf(`(function(exports, require) {
		Object.freeze(require);
		(function(){
			%v
			;
		}).apply(this);
		Object.freeze(exports);
	})`, string(bytes))

	program, err := goja.Compile(name, safeCode, true)
	if err != nil {
		return nil, err
	}
	return program, nil
}

func (r *untrustedRemoteRequire) downloadCode(origin *url.URL) ([]byte, error) {
	if !r.sameOrigin(origin) {
		return nil, errors.New("an untrusted remote require is trying to download code from a origin different from its own. A new require should have been created to do that!")
	}
	req, err := http.NewRequest("GET", origin.String(), nil)
	if err != nil {
		// TODO: rethink this, as it might leak private info to a module that we do not trust!
		return nil, err
	}
	res, err := r.httpClient.Do(req)
	if err != nil {
		// TODO: rethink this, as it might leak private info to a module that we do not trust!
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, errors.New("unexpected status from remote endpoint, expecting 200")
	}
	code, err := ioutil.ReadAll(res.Body)
	if err != nil {
		// TODO: rethink this, as it might leak private info to a module that we do not trust!
		return nil, err
	}
	return code, nil
}

func (r *untrustedRemoteRequire) sameOrigin(other *url.URL) bool {
	if r.origin == nil {
		return false
	}
	return other.Hostname() == r.origin.Hostname() &&
		other.Scheme == r.origin.Scheme
}

func computeOrigin(other *url.URL) *url.URL {
	u := new(url.URL)
	*u = *other
	u.Path = ""
	// copy user info to avoid surprises
	if other.User != nil {
		*u.User = *(other.User)
	}
	return u
}

func (r *untrustedRemoteRequire) sub(target *url.URL) *untrustedRemoteRequire {
	base := *target
	base.Path = path.Dir(base.Path)
	return &untrustedRemoteRequire{
		root:       r.root,
		origin:     r.origin,
		httpClient: r.httpClient,
		baseURL:    &base,
	}
}

func (r *untrustedRemoteRequire) absURL(relative string) *url.URL {
	base := *r.baseURL
	base.Path = path.Join(r.baseURL.Path, relative)
	return &base
}

func (r *untrustedRemoteRequire) newOrigin(otherOrigin *url.URL) *untrustedRemoteRequire {
	baseURL := *otherOrigin
	baseURL.Path = path.Dir(baseURL.Path)
	return &untrustedRemoteRequire{
		root:       r.root,
		origin:     computeOrigin(otherOrigin),
		httpClient: r.httpClient,
	}
}
