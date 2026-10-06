export class Message {
    userID;
    groupID;
    private;
    content;
    constructor(content) {
        this.content = content;
    }

    getData() {
        return {
            content: this.content,
            groupID: this.groupID || -1,
            userID: this.userID || -1,
            private: this.private
        }
    }
}