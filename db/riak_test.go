package db

import (
	"github.com/tsuru/config"
	"launchpad.net/gocheck"
	//	"reflect"
	"log"
	"strings"
	"sync"
	"testing"
	"time"
)

func Test(t *testing.T) { gocheck.TestingT(t) }

type S struct{}

type ExampleData struct {
	Field1 string `riak:"index" json:"field1"`
	Field2 int    `json:"field2"`
}

type AppRequests struct {
	id            string `json:"id"`
	node_id       string `json:"node_id"`
	node_name     string `json:"node_name"`
	appdefns_id   string `json:"appdefns_id"`
	req_type      string `json:"req_type"`
	lc_apply      string `json:"lc_apply"`
	lc_additional string `json:"lc_additional"`
	lc_when       string `json:"lc_when"`
	created_at    string `json:"created_at"`
}

var _ = gocheck.Suite(&S{})

var addr = []string{"127.0.0.1:8087"}

const bkt = "appreqs"

func (s *S) SetUpSuite(c *gocheck.C) {
	ticker.Stop()
}

func (s *S) TearDownTest(c *gocheck.C) {
	conn = make(map[string]*session)
}

func (s *S) TestOpenReconnects(c *gocheck.C) {
	log.Println("--> start [TestOpenReconnects]")
	storage, err := Open(addr, bkt)
	c.Assert(err, gocheck.IsNil)
	storage.Close()
	storage, err = Open(addr, bkt)
	defer storage.Close()
	c.Assert(err, gocheck.IsNil)
	_, err = storage.coder_client.Ping()
	c.Assert(err, gocheck.IsNil)
	log.Println("--> end   [TestOpenReconnects]")

}

func (s *S) TestOpenConnectionRefused(c *gocheck.C) {
	log.Println("--> start [TestOpenConnectionRefused]")
	storage, err := Open([]string{"127.0.0.1:68098"}, bkt)
	c.Assert(storage, gocheck.IsNil)
	c.Assert(err, gocheck.NotNil)
	log.Println("--> end   [TestOpenConnectionRefused]")

}

func (s *S) TestClose(c *gocheck.C) {
	log.Println("--> start [TestClose]")

	defer func() {
		r := recover()
		c.Check(r, gocheck.IsNil)
	}()

	storage, err := Open(addr, bkt)
	defer storage.Close()
	c.Assert(err, gocheck.IsNil)
	c.Assert(storage, gocheck.NotNil)
	_, err = storage.coder_client.Ping()
	c.Check(err, gocheck.IsNil)
	log.Println("--> end   [TestClose]")

}

func (s *S) TestConn(c *gocheck.C) {
	log.Println("--> start [TestConn]")

	config.Set("riak:url", "127.0.0.1:8087")
	defer config.Unset("riak:url")
	config.Set("riak:bucket", "appreqs")
	defer config.Unset("riak:bucket")
	storage, err := Conn("appreqs")
	defer storage.Close()
	c.Assert(storage, gocheck.NotNil)
	c.Assert(err, gocheck.IsNil)
	_, err = storage.coder_client.Ping()
	c.Check(err, gocheck.IsNil)
	log.Println("--> end   [TestConn]")

}

func (s *S) TestStore(c *gocheck.C) {
	log.Println("--> start [TestFetch]")
	config.Set("riak:url", "127.0.0.1:8087")
	defer config.Unset("riak:url")
	config.Set("riak:bucket", "appreqs")
	defer config.Unset("riak:bucket")
	storage, err := Conn("appreqs")
	defer storage.Close()
	c.Assert(storage, gocheck.NotNil)
	c.Assert(err, gocheck.IsNil)
	// Store Struct (uses coder)
	data := ExampleData{
		Field1: "ExampleData1",
		Field2: 1,
	}
	err = storage.StoreStruct("sampledata", &data)
	c.Assert(err, gocheck.IsNil)
	log.Println("--> end   [TestFetch]")
}

func (s *S) TestFetch(c *gocheck.C) {
	log.Println("--> start [TestFetch]")
	config.Set("riak:url", "127.0.0.1:8087")
	defer config.Unset("riak:url")
	config.Set("riak:bucket", "appreqs")
	defer config.Unset("riak:bucket")
	storage, err := Conn("appreqs")
	defer storage.Close()
	c.Assert(storage, gocheck.NotNil)
	c.Assert(err, gocheck.IsNil)
	out := &AppRequests{}
	err = storage.FetchStruct("sampledata", out)
	log.Println("--> value   [TestFetch] [%s]", out.node_id)
	c.Assert(err, gocheck.IsNil)
	log.Println("--> end   [TestFetch]")
}

/*func (s *S) TestUsers(c *gocheck.C) {
	storage, _ := Open("127.0.0.1:27017", "megam_storage_test")
	defer storage.Close()
	users := storage.Users()
	usersc := storage.Collection("users")
	c.Assert(users, gocheck.DeepEquals, usersc)
	c.Assert(users, HasUniqueIndex, []string{"email"})
}
*/

func (s *S) TestRetire(c *gocheck.C) {
	log.Println("--> start [TestRetire]")
	defer func() {
		if r := recover(); !c.Failed() && r == nil {
			c.Errorf("Should panic in ping, but did not!")
		}
	}()
	Open(addr, bkt)
	ky := strings.Join(addr, "::")
	sess := conn[ky]
	sess.used = sess.used.Add(-1 * 2 * period)
	conn[ky] = sess
	var ticker time.Ticker
	ch := make(chan time.Time, 1)
	ticker.C = ch
	ch <- time.Now()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		retire(&ticker)
		wg.Done()
	}()
	close(ch)
	wg.Wait()
	_, ok := conn[ky]
	c.Check(ok, gocheck.Equals, false)
	sess1 := conn[ky]
	sess1.s.Ping()
	log.Println("--> end [TestOpenReconnects]")
}
