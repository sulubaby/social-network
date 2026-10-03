CREATE TRIGGER increase_post_count
AFTER INSERT ON posts
FOR EACH ROW
BEGIN
    UPDATE profile
    SET num_of_posts = num_of_posts + 1
    WHERE user_id = NEW.user_id;
END;


CREATE TRIGGER decrease_post_count
AFTER DELETE ON posts
FOR EACH ROW
BEGIN
    UPDATE profile
    SET num_of_posts = num_of_posts - 1
    WHERE user_id = OLD.user_id;
END;


CREATE TRIGGER increase_following_count
AFTER INSERT ON user_followers
FOR EACH ROW
BEGIN
    UPDATE profile
    SET num_of_following = num_of_following + 1
    WHERE user_id = NEW.follower_id;
END;


CREATE TRIGGER increase_follower_count
AFTER INSERT ON user_followers
FOR EACH ROW
BEGIN
    UPDATE profile
    SET num_of_followers = num_of_followers + 1
    WHERE user_id = NEW.target_id;
END;


CREATE TRIGGER decrease_following_count
AFTER DELETE ON user_followers
FOR EACH ROW
BEGIN
    UPDATE profile
    SET num_of_following = num_of_following - 1
    WHERE user_id = OLD.follower_id;
END;


CREATE TRIGGER decrease_follower_count
AFTER DELETE ON user_followers
FOR EACH ROW
BEGIN
    UPDATE profile
    SET num_of_followers = num_of_followers - 1
    WHERE user_id = OLD.target_id;
END;


CREATE TRIGGER increase_members_count
AFTER INSERT ON chat_users
FOR EACH ROW
BEGIN
    UPDATE chats
    SET num_of_members = num_of_members + 1
    WHERE id = NEW.chat_id;
END;


CREATE TRIGGER decrease_members_count
AFTER DELETE ON chat_users
FOR EACH ROW
BEGIN
    UPDATE chats
    SET num_of_members = num_of_members - 1
    WHERE id = OLD.chat_id;
END;


CREATE TRIGGER increase_comment_count
AFTER INSERT ON comments
FOR EACH ROW
BEGIN
    UPDATE posts
    SET comment_count = comment_count + 1
    WHERE id = NEW.post_id;
END;


CREATE TRIGGER decrease_comment_count
AFTER DELETE ON comments
FOR EACH ROW
BEGIN
    UPDATE posts
    SET comment_count = comment_count - 1
    WHERE id = OLD.post_id;
END;


CREATE TRIGGER increase_reaction_like
AFTER INSERT ON post_reactions
FOR EACH ROW
WHEN NEW.value = 1
BEGIN
    UPDATE posts
    SET like_count = like_count + 1
    WHERE id = NEW.post_id;
END;


CREATE TRIGGER increase_reaction_dislike
AFTER INSERT ON post_reactions
FOR EACH ROW
WHEN NEW.value = -1
BEGIN
    UPDATE posts
    SET dislike_count = dislike_count + 1
    WHERE id = NEW.post_id;
END;


CREATE TRIGGER decrease_reaction_like
AFTER DELETE ON post_reactions
FOR EACH ROW
WHEN OLD.value = 1
BEGIN
    UPDATE posts
    SET like_count = like_count - 1
    WHERE id = OLD.post_id;
END;


CREATE TRIGGER decrease_reaction_dislike
AFTER DELETE ON post_reactions
FOR EACH ROW
WHEN OLD.value = -1
BEGIN
    UPDATE posts
    SET dislike_count = dislike_count - 1
    WHERE id = OLD.post_id;
END;


CREATE TRIGGER increase_reaction_comment_like
AFTER INSERT ON comments_reactions
FOR EACH ROW
WHEN NEW.value = 1
BEGIN
    UPDATE comments
    SET like_count = like_count + 1
    WHERE id = NEW.comment_id;
END;


CREATE TRIGGER increase_reaction_comment_dislike
AFTER INSERT ON comments_reactions
FOR EACH ROW
WHEN NEW.value = -1
BEGIN
    UPDATE comments
    SET dislike_count = dislike_count + 1
    WHERE id = NEW.comment_id;
END;


CREATE TRIGGER decrease_reaction_comment_like
AFTER DELETE ON comments_reactions
FOR EACH ROW
WHEN OLD.value = 1
BEGIN
    UPDATE comments
    SET like_count = like_count - 1
    WHERE id = OLD.comment_id;
END;


CREATE TRIGGER decrease_reaction_comment_dislike
AFTER DELETE ON comments_reactions
FOR EACH ROW
WHEN OLD.value = -1
BEGIN
    UPDATE comments
    SET dislike_count = dislike_count - 1
    WHERE id = OLD.comment_id;
END;