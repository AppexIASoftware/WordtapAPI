package repositories

// accessTierOrder builds the ORDER BY expression for the catalog priority
// policy: free content first, course-purchase content next and
// subscription-only content last.
//
// FIELD() is mandatory here because access_tier is stored as varchar(20), not as
// an ENUM. A plain "ORDER BY access_tier ASC" would sort alphabetically
// ("course", "free", "subscription") and push paid content to the top.
//
// The column argument must be qualified when the statement joins other tables,
// for example "lessons.access_tier".
func accessTierOrder(column string) string {
	return "FIELD(" + column + ", 'free', 'course', 'subscription')"
}
