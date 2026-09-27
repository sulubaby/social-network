package posts

import (
	"database/sql"
	"errors"
	"social/internal/models"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestPostPrivacyFiltersTheFeed(t *testing.T) {
	db := newPostTestDatabase(t)

	insertPostTestUser(t, db, 1, "author")
	insertPostTestUser(t, db, 2, "follower")
	insertPostTestUser(t, db, 3, "outsider")
	insertPostTestUser(t, db, 4, "selected")
	insertPostTestUser(t, db, 5, "pending")

	_, err := db.Exec(`
		INSERT INTO user_followers (follower_id, target_id, status)
		VALUES (2, 1, 1), (4, 1, 1), (5, 1, 0)
	`)
	if err != nil {
		t.Fatal(err)
	}

	createTestPost(t, db, models.CreatePostRequest{
		Content: "public post",
		Privacy: models.PostPrivacyPublic,
	})
	createTestPost(t, db, models.CreatePostRequest{
		Content: "followers post",
		Privacy: models.PostPrivacyFollowers,
	})
	createTestPost(t, db, models.CreatePostRequest{
		Content:             "selected post",
		Privacy:             models.PostPrivacySelected,
		SelectedFollowerIDs: []int{4, 4},
	})

	assertFeedContents(t, db, 1, "public post", "followers post", "selected post")
	assertFeedContents(t, db, 2, "public post", "followers post")
	assertFeedContents(t, db, 3, "public post")
	assertFeedContents(t, db, 4, "public post", "followers post", "selected post")
	assertFeedContents(t, db, 5, "public post")
}

func TestSelectedPostRejectsSomeoneWhoIsNotAFollower(t *testing.T) {
	db := newPostTestDatabase(t)
	insertPostTestUser(t, db, 1, "author")
	insertPostTestUser(t, db, 2, "outsider")

	_, err := CreatePost(db, 1, models.CreatePostRequest{
		Content:             "private post",
		Privacy:             models.PostPrivacySelected,
		SelectedFollowerIDs: []int{2},
	})
	if !errors.Is(err, ErrInvalidPostViewer) {
		t.Fatalf("expected ErrInvalidPostViewer, got %v", err)
	}
}

func TestSelectedPostRejectsPendingFollowRequests(t *testing.T) {
	db := newPostTestDatabase(t)
	insertPostTestUser(t, db, 1, "author")
	insertPostTestUser(t, db, 2, "pending")
	if _, err := db.Exec(`
		INSERT INTO user_followers (follower_id, target_id, status)
		VALUES (2, 1, 0)
	`); err != nil {
		t.Fatal(err)
	}

	_, err := CreatePost(db, 1, models.CreatePostRequest{
		Content:             "private post",
		Privacy:             models.PostPrivacySelected,
		SelectedFollowerIDs: []int{2},
	})
	if !errors.Is(err, ErrInvalidPostViewer) {
		t.Fatalf("pending follower error = %v, want ErrInvalidPostViewer", err)
	}
}

func TestAudienceIDsAreRejectedForNonSelectedVisibility(t *testing.T) {
	if err := ValidateSelectedIDs(models.PostPrivacyPublic, []int{42}); err == nil {
		t.Fatal("public posts should not accept a leftover private audience")
	}
}

func TestPostLikesAreIdempotentAndRespectPrivacy(t *testing.T) {
	db := newPostTestDatabase(t)
	insertPostTestUser(t, db, 1, "author")
	insertPostTestUser(t, db, 2, "follower")
	insertPostTestUser(t, db, 3, "outsider")

	publicPost, err := CreatePost(db, 1, models.CreatePostRequest{
		Content: "a public post",
		Privacy: models.PostPrivacyPublic,
	})
	if err != nil {
		t.Fatal(err)
	}

	result, err := LikePost(db, 2, publicPost.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Liked || result.LikeCount != 1 {
		t.Fatalf("first like = %+v, want liked=true and count=1", result)
	}
	feed, err := ListFeedPosts(db, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(feed) != 1 || !feed[0].Liked || feed[0].LikeCount != 1 {
		t.Fatalf("feed after like = %+v, want liked=true and count=1", feed)
	}

	result, err = LikePost(db, 2, publicPost.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Liked || result.LikeCount != 1 {
		t.Fatalf("repeat like = %+v, want no duplicate count", result)
	}

	result, err = UnlikePost(db, 2, publicPost.ID)
	if err != nil {
		t.Fatal(err)
	}
	if result.Liked || result.LikeCount != 0 {
		t.Fatalf("unlike = %+v, want liked=false and count=0", result)
	}
	feed, err = ListFeedPosts(db, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(feed) != 1 || feed[0].Liked || feed[0].LikeCount != 0 {
		t.Fatalf("feed after unlike = %+v, want liked=false and count=0", feed)
	}

	result, err = UnlikePost(db, 2, publicPost.ID)
	if err != nil {
		t.Fatal(err)
	}
	if result.Liked || result.LikeCount != 0 {
		t.Fatalf("repeat unlike = %+v, want no negative count", result)
	}

	followersPost, err := CreatePost(db, 1, models.CreatePostRequest{
		Content: "a followers-only post",
		Privacy: models.PostPrivacyFollowers,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = LikePost(db, 3, followersPost.ID); !errors.Is(err, ErrPostNotVisible) {
		t.Fatalf("outsider like error = %v, want ErrPostNotVisible", err)
	}
}

func TestFeedCanLoadOnePageAtATime(t *testing.T) {
	db := newPostTestDatabase(t)
	insertPostTestUser(t, db, 1, "author")

	for _, content := range []string{"first", "second", "third"} {
		createTestPost(t, db, models.CreatePostRequest{
			Content: content,
			Privacy: models.PostPrivacyPublic,
		})
	}

	page, err := ListFeedPosts(db, 1, 3, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(page) != 3 {
		t.Fatalf("first page length = %d, expected 3", len(page))
	}

	olderPage, err := ListFeedPosts(db, 1, 3, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(olderPage) != 0 {
		t.Fatalf("older page length = %d, expected 0", len(olderPage))
	}
}

func TestProfilePostsFollowPrivacyAndPaging(t *testing.T) {
	db := newPostTestDatabase(t)
	insertPostTestUser(t, db, 1, "author")
	insertPostTestUser(t, db, 2, "follower")
	insertPostTestUser(t, db, 3, "outsider")
	if _, err := db.Exec(`INSERT INTO profile (user_id) VALUES (1), (2), (3)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO user_followers (follower_id, target_id, status) VALUES (2, 1, 1)`); err != nil {
		t.Fatal(err)
	}

	createTestPost(t, db, models.CreatePostRequest{Content: "public one", Privacy: models.PostPrivacyPublic})
	createTestPost(t, db, models.CreatePostRequest{Content: "followers one", Privacy: models.PostPrivacyFollowers})
	createTestPost(t, db, models.CreatePostRequest{Content: "public two", Privacy: models.PostPrivacyPublic})

	count := func(viewerID, limit, offset int) int {
		t.Helper()
		posts, err := ListProfilePosts(db, viewerID, 1, limit, offset)
		if err != nil {
			t.Fatal(err)
		}
		for _, post := range posts {
			if post.UserID != 1 {
				t.Fatalf("profile of user 1 returned a post from user %d", post.UserID)
			}
		}
		return len(posts)
	}

	if got := count(1, 10, 0); got != 3 {
		t.Fatalf("author sees %d of their own posts, expected 3", got)
	}
	if got := count(2, 10, 0); got != 3 {
		t.Fatalf("follower sees %d posts, expected 3", got)
	}
	if got := count(3, 10, 0); got != 2 {
		t.Fatalf("outsider sees %d posts, expected only the 2 public ones", got)
	}
	if got := count(2, 2, 2); got != 1 {
		t.Fatalf("second page has %d posts, expected 1", got)
	}

	if _, err := db.Exec(`UPDATE profile SET is_private = 1 WHERE user_id = 1`); err != nil {
		t.Fatal(err)
	}
	if got := count(3, 10, 0); got != 0 {
		t.Fatalf("outsider sees %d posts on a private profile, expected 0", got)
	}
	if got := count(2, 10, 0); got != 3 {
		t.Fatalf("follower sees %d posts on a private profile, expected 3", got)
	}
}

func createTestPost(t *testing.T, db *sql.DB, request models.CreatePostRequest) {
	t.Helper()
	if _, err := CreatePost(db, 1, request); err != nil {
		t.Fatal(err)
	}
}

func assertFeedContents(t *testing.T, db *sql.DB, viewerID int, expected ...string) {
	t.Helper()

	posts, err := ListFeedPosts(db, viewerID)
	if err != nil {
		t.Fatal(err)
	}
	if len(posts) != len(expected) {
		t.Fatalf("viewer %d received %d posts, expected %d", viewerID, len(posts), len(expected))
	}

	found := make(map[string]bool)
	for _, post := range posts {
		found[post.Content] = true
	}
	for _, content := range expected {
		if !found[content] {
			t.Fatalf("viewer %d did not receive %q", viewerID, content)
		}
	}
}

func newPostTestDatabase(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })

	_, err = db.Exec(`
		PRAGMA foreign_keys = ON;

		CREATE TABLE user (
			id INTEGER PRIMARY KEY,
			email TEXT NOT NULL UNIQUE,
			username TEXT NOT NULL UNIQUE,
			first_name TEXT NOT NULL,
			last_name TEXT NOT NULL,
			dob DATE NOT NULL,
			password TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE groups (
			id INTEGER PRIMARY KEY,
			title TEXT NOT NULL,
			description TEXT NOT NULL
		);

		CREATE TABLE posts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			type TEXT NOT NULL DEFAULT 'post',
			title TEXT NOT NULL DEFAULT '',
			content TEXT NOT NULL,
			image_path TEXT NOT NULL DEFAULT '',
			user_id INTEGER NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			group_id INTEGER,
			privacy TEXT NOT NULL DEFAULT 'public',
			location TEXT,
			like_count INTEGER NOT NULL DEFAULT 0,
			dislike_count INTEGER NOT NULL DEFAULT 0,
			comment_count INTEGER NOT NULL DEFAULT 0,
			FOREIGN KEY (user_id) REFERENCES user(id),
			FOREIGN KEY (group_id) REFERENCES groups(id)
		);

		CREATE TABLE user_followers (
			follower_id INTEGER NOT NULL,
			target_id INTEGER NOT NULL,
			status INTEGER NOT NULL DEFAULT 1,
			PRIMARY KEY (follower_id, target_id),
			FOREIGN KEY (follower_id) REFERENCES user(id),
			FOREIGN KEY (target_id) REFERENCES user(id)
		);

		CREATE TABLE profile (
			user_id INTEGER PRIMARY KEY,
			avatar_path TEXT,
			is_private INTEGER NOT NULL DEFAULT 0,
			FOREIGN KEY (user_id) REFERENCES user(id)
		);

		CREATE TABLE post_viewers (
			post_id INTEGER NOT NULL,
			viewer_id INTEGER NOT NULL,
			PRIMARY KEY (post_id, viewer_id),
			FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE,
			FOREIGN KEY (viewer_id) REFERENCES user(id) ON DELETE CASCADE
		);

		CREATE TABLE post_reactions (
			user_id INTEGER NOT NULL,
			post_id INTEGER NOT NULL,
			value INTEGER NOT NULL,
			PRIMARY KEY (user_id, post_id),
			FOREIGN KEY (user_id) REFERENCES user(id) ON DELETE CASCADE,
			FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE
		);

		CREATE TRIGGER increase_reaction_like
		AFTER INSERT ON post_reactions
		WHEN NEW.value = 1
		BEGIN
			UPDATE posts SET like_count = like_count + 1 WHERE id = NEW.post_id;
		END;

		CREATE TRIGGER decrease_reaction_like
		AFTER DELETE ON post_reactions
		WHEN OLD.value = 1
		BEGIN
			UPDATE posts SET like_count = like_count - 1 WHERE id = OLD.post_id;
		END;
	`)
	if err != nil {
		t.Fatal(err)
	}

	return db
}

func insertPostTestUser(t *testing.T, db *sql.DB, id int, username string) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO user (id, email, username, first_name, last_name, dob, password)
		VALUES (?, ?, ?, 'Test', 'User', '2000-01-01', 'password')
	`, id, username+"@orbit.test", username)
	if err != nil {
		t.Fatal(err)
	}
}
