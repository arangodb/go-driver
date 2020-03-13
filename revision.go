package driver

import (
	"context"
	"fmt"
	"path"
)

// RevisionInt64 is representation of '_rev' string value as an int64 number
type RevisionInt64 struct {
	Rev  int64
	RevS string // TODO once I know how to encode Rev into string then It will be deleted
}

// RevisionMinMax is an array of two Revisions which create range of them
type RevisionMinMax [2]RevisionInt64

// Revisions is a slice of Revisions
type Revisions []RevisionInt64

// RevisionTreeNode is a bucket (leaf) in Merkle tree with hashed Revisions and with count of documents in the bucket
type RevisionTreeNode struct {
	Hash  string `json:"hash"`
	Count int64  `json:"count,int"`
}

// RevisionTree is a list of Revisions in a Merkle tree
type RevisionTree struct {
	Version  int                `json:"version"`
	RangeMin RevisionInt64      `json:"rangeMin,string"`
	RangeMax RevisionInt64      `json:"rangeMax,string""`
	Nodes    []RevisionTreeNode `json:"nodes"`
}

// UnmarshalJSON parses string revision document into int64 number
func (n *RevisionInt64) UnmarshalJSON(source []byte) (err error) {
	// TODO alphapet should be a map or use base64.NewEcode + alphabet
	alphabet := [64]byte{'-', '_',
		'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z',
		'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n', 'o', 'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z',
		'0', '1', '2', '3', '4', '5', '6', '7', '8', '9'}

	var re int64
	for _, s := range source {
		for k, a := range alphabet {
			if a == s {
				re = re*64 + int64(k)
				break
			}
		}
	}

	n.Rev = re
	n.RevS = string(source)

	return nil
}

// MarshalJSON converts int64 into string revision
func (n *RevisionInt64) MarshalJSON() ([]byte, error) {
	var t int64 = n.Rev
	index := 11

	//"_aLSfdI----"
	//1661074099696304128
	alphabet := [64]byte{'-', '_',
		'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z',
		'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n', 'o', 'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z',
		'0', '1', '2', '3', '4', '5', '6', '7', '8', '9'}

	string1 := make([]byte, 0)

	for t > 0 {
		index--
		in := uint8(t & 0x3f)

		string1 = append(string1, alphabet[in])
		t >>= 6
	}

	fmt.Printf("%s", string(string1))
	return []byte(n.RevS), nil
}

/*
  static std::pair<size_t, size_t> encodeTimeStamp(uint64_t t, char* r) {
    size_t pos = 11;
    while (t > 0) {
      r[--pos] = encodeTable[static_cast<uint8_t>(t & 0x3ful)];
      t >>= 6;
    }
    return std::make_pair(pos, 11 - pos);
  }

*/

// GetRevisionTree retrieves the Revision tree (Merkel tree) associated with the collection.
func (c *client) GetRevisionTree(ctx context.Context, db Database, batchId, collection string) (RevisionTree, error) {

	req, err := c.conn.NewRequest("GET", path.Join("_db", db.Name(), "_api/replication/revisions/tree"))
	if err != nil {
		return RevisionTree{}, WithStack(err)
	}

	req = req.SetQuery("batchId", batchId)
	req = req.SetQuery("collection", collection)

	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return RevisionTree{}, WithStack(err)
	}

	if err := resp.CheckStatus(200); err != nil {
		return RevisionTree{}, WithStack(err)
	}

	var tree RevisionTree
	if err := resp.ParseBody("", &tree); err != nil {
		return RevisionTree{}, WithStack(err)
	}

	return tree, nil
}

// GetRevisionsByRanges retrieves the revision IDs of documents within requested ranges.
func (c *client) GetRevisionsByRanges(ctx context.Context, db Database, batchId, collection string,
	minMaxRevision []RevisionMinMax, resume *RevisionInt64) ([]Revisions, error) {

	req, err := c.conn.NewRequest("PUT", path.Join("_db", db.Name(), "_api/replication/revisions/ranges"))
	if err != nil {
		return nil, WithStack(err)
	}

	req = req.SetQuery("batchId", batchId)
	req = req.SetQuery("collection", collection)
	if resume != nil {
		req = req.SetQuery("resume", resume.RevS)
	}

	req, err = req.SetBody(minMaxRevision)
	if err != nil {
		return nil, WithStack(err)
	}

	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return nil, WithStack(err)
	}

	if err := resp.CheckStatus(200); err != nil {
		return nil, WithStack(err)
	}

	ranges := make([]Revisions, 0)
	if err := resp.ParseBody("ranges", &ranges); err != nil {
		return nil, WithStack(err)
	}

	return ranges, nil
}

// GetRevisionDocuments retrieves documents by revision.
func (c *client) GetRevisionDocuments(ctx context.Context, db Database, batchId, collection string,
	revisions Revisions) ([]map[string]interface{}, error) {

	req, err := c.conn.NewRequest("PUT", path.Join("_db", db.Name(), "_api/replication/revisions/documents"))
	if err != nil {
		return nil, WithStack(err)
	}

	req = req.SetQuery("batchId", batchId)
	req = req.SetQuery("collection", collection)

	req, err = req.SetBody(revisions)
	if err != nil {
		return nil, WithStack(err)
	}

	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return nil, WithStack(err)
	}

	if err := resp.CheckStatus(200); err != nil {
		return nil, WithStack(err)
	}

	arrayResponse, err := resp.ParseArrayBody()
	if err != nil {
		return nil, WithStack(err)
	}

	documents := make([]map[string]interface{}, 0, len(arrayResponse))
	for _, a := range arrayResponse {
		document := map[string]interface{}{}
		if err = a.ParseBody("", &document); err != nil {
			return nil, WithStack(err)
		}
		documents = append(documents, document)
	}

	return documents, nil
}

// TODO implement Marshal and UnMarshal for vst
