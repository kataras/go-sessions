package sessions

import (
	"encoding/json"
	"github.com/gavv/httpexpect"
	"github.com/kataras/go-errors"
	"github.com/kataras/go-serializer"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
)

var errReadBody = errors.New("While trying to read %s from the request body. Trace %s")

// ReadJSON reads JSON from request's body
func ReadJSON(jsonObject interface{}, req *http.Request) error {
	b, err := ioutil.ReadAll(req.Body)
	if err != nil && err != io.EOF {
		return err
	}
	decoder := json.NewDecoder(strings.NewReader(string(b)))
	err = decoder.Decode(jsonObject)

	if err != nil && err != io.EOF {
		return errReadBody.Format("JSON", err.Error())
	}
	return nil
}

func TestSessionsNetHTTP(t *testing.T) {
	t.Parallel()

	values := map[string]interface{}{
		"Name":   "go-sessions",
		"Days":   "1",
		"Secret": "dsads£2132215£%%Ssdsa",
	}

	setHandler := http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		vals := make(map[string]interface{}, 0)
		if err := ReadJSON(&vals, req); err != nil {
			t.Fatalf("Cannot readjson. Trace %s", err.Error())
		}
		sess := Start(res, req)
		for k, v := range vals {
			sess.Set(k, v)
		}

		res.WriteHeader(http.StatusOK)
	})
	http.Handle("/set/", setHandler)

	writeValues := func(res http.ResponseWriter, req *http.Request) {
		sess := Start(res, req)
		sessValues := sess.GetAll()

		//t.Logf("sessValues length: %d", len(sessValues))

		result, err := serializer.Serialize("application/json", sessValues)
		if err != nil {
			t.Fatalf("While serialize the session values: %s", err.Error())
		}

		res.Header().Set("Content-Type", "application/json")
		res.Write(result)
	}

	getHandler := http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		writeValues(res, req)
	})
	http.Handle("/get/", getHandler)

	clearHandler := http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		sess := Start(res, req)
		sess.Clear()
		writeValues(res, req)
	})
	http.Handle("/clear/", clearHandler)

	destroyHandler := http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		Destroy(res, req)
		writeValues(res, req)
		res.WriteHeader(http.StatusOK)
		// the cookie and all values should be empty
	})
	http.Handle("/destroy/", destroyHandler)

	afterDestroyHandler := http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(http.StatusOK)
	})
	// request cookie should be empty
	http.Handle("/after_destroy/", afterDestroyHandler)

	testConfiguration := httpexpect.Config{
		BaseURL: "http://localhost:8080",
		Client: &http.Client{
			Transport: httpexpect.NewBinder(http.DefaultServeMux),
			Jar:       httpexpect.NewJar(),
		},
		Reporter: httpexpect.NewAssertReporter(t),
	}

	e := httpexpect.WithConfig(testConfiguration)

	e.POST("/set/").WithJSON(values).Expect().Status(http.StatusOK).Cookies().NotEmpty()
	e.GET("/get/").Expect().Status(http.StatusOK).JSON().Object().Equal(values)

	// test destory which also clears first
	d := e.GET("/destroy/").Expect().Status(http.StatusOK)
	d.JSON().Object().Empty()
	e.GET("/after_destroy/").Expect().Status(http.StatusOK).Cookies().Empty()
	// set and clear again
	e.POST("/set/").WithJSON(values).Expect().Status(http.StatusOK).Cookies().NotEmpty()
	e.GET("/clear/").Expect().Status(http.StatusOK).JSON().Object().Empty()
}
