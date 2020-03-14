package driver

import (
	"context"
	"path"
)

// RevisionInt64 is representation of '_rev' string value as an int64 number
type RevisionInt64 int64

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
	RangeMax RevisionInt64      `json:"rangeMax,string"`
	Nodes    []RevisionTreeNode `json:"nodes"`
}

var (
	revisionEncodingTable = [64]byte{'-', '_', 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N',
		'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z', 'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k',
		'l', 'm', 'n', 'o', 'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z', '0', '1', '2', '3', '4', '5', '6', '7',
		'8', '9'}
	revisionDecodingTable = [256]byte{
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, //   0 - 15
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, //  16 - 31
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, //  32 - 47 (here is the '-' on 45 place)
		54, 55, 56, 57, 58, 59, 60, 61, 62, 63, 0, 0, 0, 0, 0, 0, //  48 - 63
		0, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, //  64 - 79
		17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 0, 0, 0, 0, 1, //  80 - 95
		0, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38, 39, 40, 41, 42, //  96 - 111
		43, 44, 45, 46, 47, 48, 49, 50, 51, 52, 53, 0, 0, 0, 0, 0, // 112 - 127
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, // 128 - 143
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, // 144 - 159
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, // 160 - 175
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, // 176 - 191
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, // 192 - 207
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, // 208 - 223
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, // 224 - 239
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, // 240 - 255
	}
)

// UnmarshalJSON parses string revision document into int64 number
func (n *RevisionInt64) UnmarshalJSON(source []byte) (err error) {
	var t int64

	for _, s := range source {
		if s == '"' {
			continue
		}
		t = t*64 + int64(revisionDecodingTable[s])
	}
	*n = RevisionInt64(t)
	return nil
}

// MarshalJSON converts int64 into string revision
func (n *RevisionInt64) MarshalJSON() ([]byte, error) {
	if *n == 0 {
		return []byte{'"', '"'}, nil // return an empty string
	}

	var result [16]byte
	t := *n
	result[15] = '"'
	index := 14

	for t > 0 {
		result[index] = revisionEncodingTable[uint8(t&0x3f)]
		t >>= 6
		index--
	}
	result[index] = '"'

	return result[index:], nil
}

//func (n *RevisionInt64) UnmarshalVPack(slice velocypack.Slice) error {
//	source, err := slice.GetString()
//	if err != nil {
//		return err
//	}
//
//	var t int64
//
//	for _, s := range source {
//		if s == '"' {
//			continue
//		}
//		t = t*64 + int64(revisionDecodingTable[s])
//	}
//	*n = RevisionInt64(t)
//	return nil
//}
//
//func (n *RevisionInt64) MarshalVPack() (velocypack.Slice, error) {
//	var b velocypack.Builder
//
//	var result [16]byte
//	t := *n
//
//	index := 15
//
//	for t > 0 {
//		result[index] = revisionEncodingTable[uint8(t&0x3f)]
//		t >>= 6
//		index--
//	}
//
//	if err := b.AddValue(velocypack.NewStringValue(string(result[index+1:]))); err != nil {
//		return nil, err
//	}
//
//	res, _ := b.Slice()
//	dupa := make([]byte, 0, len(res))
//
//	for _, j := range res {
//
//		dupa = append(dupa, j)
//	}
//	b.Close()
//
//	return dupa, nil
//}

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
		bytes, _ := resume.MarshalJSON()
		req = req.SetQuery("resume", string(bytes))
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
