package readas

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"github.com/writeas/activity/streams"
	"github.com/writeas/go-webfinger"
	"github.com/writeas/httpsig"
	"github.com/writeas/impart"
	"github.com/writeas/web-core/activitypub"
	"github.com/writeas/web-core/activitystreams"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"strings"
)

var (
	verifier *httpsig.Verifier
)

func initFederation(app *app) {
	verifier = httpsig.NewSigHeaderVerifier(keyGetter{app})
}

type keyGetter struct {
	app *app
}

func (kg keyGetter) GetKey(id string) interface{} {
	k, err := kg.app.getActorKey(id)
	if err != nil {
		logError("Unable to get key: %v", err)
		return nil
	}
	pubKey, err := activitypub.DecodePublicKey(k)
	if err != nil {
		logError("Unable to decode key: %v", err)
		return err
	}
	return pubKey
}

func verifyRequest(app *app, r *http.Request) error {
	return verifier.Verify(r)
}

type WebfingerResult struct {
	ActorIRI string
	Username string
	Host     string
}

func makeActivityPost(p *activitystreams.Person, url string, m interface{}) error {
	logInfo("POST %s", url)
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}

	r, _ := http.NewRequest("POST", url, bytes.NewBuffer(b))
	r.Header.Add("Content-Type", "application/activity+json")
	r.Header.Set("User-Agent", userAgent)
	h := sha256.New()
	h.Write(b)
	r.Header.Add("Digest", "SHA-256="+base64.StdEncoding.EncodeToString(h.Sum(nil)))

	// Sign using the 'Signature' header
	privKey, err := activitypub.DecodePrivateKey(p.GetPrivKey())
	if err != nil {
		return err
	}
	signer := httpsig.NewSigner(p.PublicKey.ID, privKey, httpsig.RSASHA256, []string{"(request-target)", "date", "host", "digest"})
	err = signer.SignSigHeader(r)
	if err != nil {
		logError("Can't sign: %v", err)
	}

	dump, err := httputil.DumpRequestOut(r, true)
	if err != nil {
		logError("Can't dump: %v", err)
	} else {
		logInfo("%s", dump)
	}

	resp, err := http.DefaultClient.Do(r)
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}

	if resp == nil {
		logInfo("No status or response.")
	} else {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		logInfo("Status  : %s", resp.Status)
		logInfo("Response: %s", body)
	}

	return nil
}

func resolveIRI(url string) ([]byte, error) {
	logInfo("GET %s", url)

	r, _ := http.NewRequest("GET", url, nil)
	r.Header.Add("Accept", "application/activity+json")
	r.Header.Set("User-Agent", userAgent)

	dump, err := httputil.DumpRequestOut(r, true)
	if err != nil {
		logError("Can't dump: %v", err)
	} else {
		logInfo("%s", dump)
	}

	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		return nil, err
	}
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	logInfo("Status  : %s", resp.Status)
	logInfo("Response: %s", body)

	return body, nil
}

func doWebfinger(host, username string) (*WebfingerResult, error) {
	url := "https://" + host + "/.well-known/webfinger?resource=acct:" + username + "@" + host
	logInfo("Webfinger: %s", url)

	r, _ := http.NewRequest("GET", url, nil)
	r.Header.Set("User-Agent", userAgent)

	dump, err := httputil.DumpRequestOut(r, true)
	if err != nil {
		logError("Can't dump: %v", err)
	} else {
		logInfo("%s", dump)
	}

	resp, err := http.DefaultClient.Do(r)
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	logInfo("Status  : %s", resp.Status)
	logInfo("Response: %s", body)

	if resp.StatusCode != 200 {
		return nil, impart.HTTPError{resp.StatusCode, ""}
	}

	resource := webfinger.Resource{}
	if err := json.Unmarshal(body, &resource); err != nil {
		logError("Unable to unmarshal webfinger: %v", err)
	}

	var actorIRI string
	// TODO: ensure subject matches
	// TODO: store text/html URL here?
	for _, l := range resource.Links {
		if l.Type != "application/activity+json" || l.Rel != "self" {
			continue
		}
		actorIRI = l.HRef
		break
	}
	subj := resource.Subject
	if strings.HasPrefix(subj, "acct:") {
		subj = subj[len("acct:"):]
	}
	handleItems := strings.Split(subj, "@")

	return &WebfingerResult{
		Username: handleItems[0],
		Host:     handleItems[1],
		ActorIRI: actorIRI,
	}, nil
}

func fetchActor(app *app, actorIRI string) (*activitystreams.Person, *User, error) {
	logInfo("Fetching actor %s locally", actorIRI)
	actor := &activitystreams.Person{}
	remoteUser, err := app.getActor(actorIRI)
	if err != nil {
		if iErr, ok := err.(impart.HTTPError); ok {
			if iErr.Status == http.StatusNotFound {
				// Fetch remote actor
				logInfo("Not found; fetching actor %s remotely", actorIRI)
				actorResp, err := resolveIRI(actorIRI)
				if err != nil {
					logError("Unable to get actor! %v", err)
					return nil, nil, impart.HTTPError{http.StatusInternalServerError, "Couldn't fetch actor."}
				}
				if err := json.Unmarshal(actorResp, &actor); err != nil {
					// FIXME: Hubzilla has an object for the Actor's url: cannot unmarshal object into Go struct field Person.url of type string
					logError("Unable to unmarshal actor! %v", err)
					return nil, nil, impart.HTTPError{http.StatusInternalServerError, "Couldn't parse actor."}
				}
			} else {
				return nil, nil, err
			}
		} else {
			return nil, nil, err
		}
	} else {
		actor = remoteUser.AsPerson()
	}
	// TODO: don't return all three; just (*User, error)
	return actor, remoteUser, nil
}

// TODO: rename this to something better; it doesn't just fetch, but also adds posts
func fetchActorOutbox(app *app, outbox string) error {
	logInfo("Fetching actor outbox: " + outbox)
	outRes, err := resolveIRI(outbox)
	if err != nil {
		logError("Unable to get outbox! %v", err)
		return impart.HTTPError{http.StatusInternalServerError, "Couldn't fetch outbox."}
	}
	coll := &activitystreams.OrderedCollection{}
	if err := json.Unmarshal(outRes, &coll); err != nil {
		logError("Unable to unmarshal outbox! %v", err)
		return impart.HTTPError{http.StatusInternalServerError, "Couldn't parse outbox."}
	}
	if coll.First == "" {
		// Not the OrderedCollection we were expecting, so quit now and don't fetch any posts
		// TODO: parse the OrderedCollection `items` property and import those posts
		return nil
	}

	logInfo("Fetching actor outbox page: " + coll.First)
	outPageRes, err := resolveIRI(coll.First)
	if err != nil {
		logError("Unable to get outbox page! %v", err)
		return impart.HTTPError{http.StatusInternalServerError, "Couldn't fetch outbox page."}
	}

	var collPageMap map[string]interface{}
	if err := json.Unmarshal(outPageRes, &collPageMap); err != nil {
		return err
	}
	res := &streams.Resolver{
		OrderedCollectionPageCallback: func(p *streams.OrderedCollectionPage) error {
			itemsLen := p.LenOrderedItems()
			logInfo("Ordered items: %d", itemsLen)
			if itemsLen > 0 {
				// Add posts in reverse order they're listed, since they're in reverse-chronological order
				for i := itemsLen - 1; i >= 0; i-- {
					_, err = p.ResolveOrderedItems(&streams.Resolver{
						CreateCallback: func(f *streams.Create) error {
							// Create post in database
							return handleCreateCallback(app, f)
						},
					}, i)
					if err != nil {
						logError("Unable to resolve item: %v", err)
					}
				}
			}
			return nil
		},
	}
	if err := res.Deserialize(collPageMap); err != nil {
		logError("Unable to resolve OrderedCollectionPage: %v", err)
		return err
	}

	return nil
}
