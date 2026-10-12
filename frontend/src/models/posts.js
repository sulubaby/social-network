// Reaction is what we send when liking or disliking, value is 1 or -1
export class Reaction {
    value;
    userID;
    postID;

    constructor(value, postID) {
        this.value = value;
        this.postID = postID;
    }

    getData() {
        return {
            value: this.value,
            userID: this.userID,
            postID: this.postID
        }
    }
}

// Comment is one comment in the list, the last three fields are only for the ui
export class Comment {
    constructor(
        id,
        content,
        postId,
        replyTo,
        votes,
        createdAt,
        user,
        replies = 0,
        imagePath = ''
    ) {
        this.ID = id;
        this.content = content;
        this.postId = postId;
        this.replyTo = replyTo;
        this.votes = votes;
        this.createdAt = createdAt;
        this.user = user;
        this.replies = replies;
        this.imagePath = imagePath;
        // replies are loaded only when the user opens them
        this.loadedReplies = null;
        this.showReplies = false;
        this.pending = false;
    }
}

// createComment turns the server json into a Comment, with defaults for missing fields
export function createComment(data) {
    return new Comment(
        data.id,
        data.content,
        data.postId,
        data.replyTo ?? null,
        data.votes ?? 0,
        data.createdAt,
        data.user,
        data.replies ?? 0,
        data.imagePath ?? ''
    );
}

// Post has the same fields the server sends for a post card
export class Post {
    constructor(
        ReactionValue = 0,
        allowComments = false,
        avatarPath = '',
        content = '',
        createdAt = '',
        firstName = '',
        groupId = 0,
        id = 0,
        imagePath = '',
        lastName = '',
        location = '',
        relationship = '',
        taggedPeople = [],
        userId = 0,
        username = '',
        visibility = '',
        visibilityUser = '',
        likeCount = 0,
        disLikeCount = 0
    ) {
        this.ReactionValue = ReactionValue
        this.allowComments = allowComments
        this.avatarPath = avatarPath
        this.content = content
        this.createdAt = createdAt
        this.firstName = firstName
        this.groupId = groupId
        this.id = id
        this.imagePath = imagePath
        this.lastName = lastName
        this.location = location
        this.relationship = relationship
        this.taggedPeople = taggedPeople
        this.userId = userId
        this.username = username
        this.visibility = visibility
        this.visibilityUser = visibilityUser
        this.likeCount = likeCount
        this.disLikeCount = disLikeCount
    }
}

// Posts is a small list helper for adding and removing posts
export class Posts {
    constructor(posts = []) {
        this.posts = posts
    }

    addPost(post) {
        this.posts.push(post)
    }

    removePost(postId) {
        this.posts = this.posts.filter(post => post.id !== postId)
    }

    getPost(postId) {
        return this.posts.find(post => post.id === postId)
    }

    clear() {
        this.posts = []
    }
}
