package twitterclient

// API Base Configuration
const (
	API_HOST = "https://x.com"
)

// GraphQL Endpoint Constants
const (
	// User-related endpoints
	GRAPHQL_USER_BY_REST_ID     = "/i/api/graphql/CO4_gU4G_MRREoqfiTh6Hg/UserByRestId"
	GRAPHQL_USER_BY_SCREEN_NAME = "/i/api/graphql/xmU6X_CKVnQ5lSrCbAmJsg/UserByScreenName"
	GRAPHQL_USER_MEDIA          = "/i/api/graphql/MOLbHrtk8Ovu7DUNOLcXiA/UserMedia"
	GRAPHQL_FOLLOWING           = "/i/api/graphql/7FEKOPNAvxWASt6v9gfCXw/Following"
	GRAPHQL_LIKES               = "/i/api/graphql/aeJWz--kknVBOl7wQ7gh7Q/Likes"

	// List-related endpoints
	GRAPHQL_LIST_BY_REST_ID = "/i/api/graphql/ZMQOSpxDo0cP5Cdt8MgEVA/ListByRestId"
	GRAPHQL_LIST_MEMBERS    = "/i/api/graphql/3dQPyRyAj6Lslp4e0ClXzg/ListMembers"

	// Legacy API endpoints
	API_FRIENDSHIPS_CREATE = "/i/api/1.1/friendships/create.json"
)

// Response Path Constants
const (
	INST_PATH_USER_MEDIA    = "data.user.result.timeline_v2.timeline.instructions"
	INST_PATH_USER_TIMELINE = "data.user.result.timeline.timeline.instructions"
	INST_PATH_LIST_MEMBERS  = "data.list.members_timeline.timeline.instructions"
)

// Default Values
const (
	DEFAULT_PAGE_SIZE_FOR_TWEETS = 100
	DEFAULT_MEMBERS_PAGE_SIZE    = 200
	AVG_TWEETS_PER_PAGE          = 70
)

const AvgTweetsPerPage = 70
