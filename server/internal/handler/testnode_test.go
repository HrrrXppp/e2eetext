package handler

import "github.com/ekhrunov/messenger/server/internal/node"

const testNodeID = "99999999-9999-9999-9999-999999999999"

func testNode() node.Registry {
	return node.Registry{ID: testNodeID}
}

func scopedID(localID string) string {
	return testNode().ScopeID(localID)
}
