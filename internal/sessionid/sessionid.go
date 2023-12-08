package sessionid

import (
	crand "crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"hash/fnv"
	"math/rand"
	"net/http"
	"os"
	"sync"

	"github.com/pyroscope-io/pyroscope-lambda-extension/internal/flameql"
)

const LabelName = "__session_id__"

func InjectToRequest(sessionID string, r *http.Request) {
	parsed, err := flameql.ParseKey(r.URL.Query().Get("name"))
	if err != nil {
		// This is an invalid request, but we defer to the backend.
		return
	}
	if _, ok := parsed.Labels()[LabelName]; !ok {
		parsed.Add(LabelName, sessionID)
		q := r.URL.Query()
		q.Set("name", parsed.Normalized())
		r.URL.RawQuery = q.Encode()
	}
}

type ID uint64

func (s ID) String() string {
	var b [8]byte
	binary.LittleEndian.PutUint64(b[:], uint64(s))
	return hex.EncodeToString(b[:])
}

func New() ID { return globalSessionIDGenerator.newSessionID() }

var globalSessionIDGenerator = newSessionIDGenerator()

type sessionIDGenerator struct {
	sync.Mutex
	src *rand.Rand
}

func (gen *sessionIDGenerator) newSessionID() ID {
	var b [8]byte
	gen.Lock()
	_, _ = gen.src.Read(b[:])
	gen.Unlock()
	return ID(binary.LittleEndian.Uint64(b[:]))
}

func newSessionIDGenerator() *sessionIDGenerator {
	s, ok := sessionIDHostSeed()
	if !ok {
		s = sessionIDRandSeed()
	}
	return &sessionIDGenerator{src: rand.New(rand.NewSource(s))}
}

func sessionIDRandSeed() int64 {
	var rndSeed int64
	_ = binary.Read(crand.Reader, binary.LittleEndian, &rndSeed)
	return rndSeed
}

var hostname = os.Hostname

func sessionIDHostSeed() (int64, bool) {
	v, err := hostname()
	if err != nil {
		return 0, false
	}
	h := fnv.New64a()
	_, _ = h.Write([]byte(v))
	return int64(h.Sum64()), true
}
