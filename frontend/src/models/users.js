export class Users {
    ID;
    firstName;
    lastName;
    username;
    email;
    DOB;
    isPrivate;
    Profile;

    constructor(id, firstName, lastName, username, email, DOB, isPrivate, Profile) {
        this.ID = id;
        this.firstName = firstName;
        this.lastName = lastName;
        this.username = username;
        this.email = email;
        this.DOB = DOB;
        this.isPrivate = isPrivate;
        this.Profile = Profile;
    }

    getData() {
        return {
            ID: this.ID,
            firstName: this.firstName,
            lastName: this.lastName,
            username: this.username,
            email: this.email,
            DOB: this.DOB,
            isPrivate: this.isPrivate,
            Profile: this.Profile
        };
    }

    getID() { return this.ID; }
    setID(id) { this.ID = id; }

    getFirstName() { return this.firstName; }
    setFirstName(firstName) { this.firstName = firstName; }

    getLastName() { return this.lastName; }
    setLastName(lastName) { this.lastName = lastName; }

    getUsername() { return this.username; }
    setUsername(username) { this.username = username; }

    getEmail() { return this.email; }
    setEmail(email) { this.email = email; }

    getDOB() { return this.DOB; }
    setDOB(DOB) { this.DOB = DOB; }

    getIsPrivate() { return this.isPrivate; }
    setIsPrivate(isPrivate) { this.isPrivate = isPrivate; }

    getProfile() { return this.Profile; }
    setProfile(Profile) { this.Profile = Profile; }
}

export class Profile {
    numOfFollowing = 0;
    numOfFollowers = 0;
    numOfPosts = 0;
    avatar = '';
    About = null;
    following = [];
    followers = [];
    friends = [];
    show = true;
    constructor(numOfFollowing, numOfFollowers, numOfPosts, avatar, About, following = [], followers = [], friends = []) {
        this.numOfFollowing = numOfFollowing;
        this.numOfFollowers = numOfFollowers;
        this.numOfPosts = numOfPosts;
        this.avatar = avatar;
        this.About = About;
        this.following = following;
        this.followers = followers;
        this.friends = friends;
    }
}

export class About {
    bio;
    work;
    education;
    travel;
    interests;
    hobbies;
    website;
    linkedin;
    instagram;
    twitter;

    constructor(bio, work, education, travel, interests, hobbies, website, linkedin, instagram, twitter) {
        this.bio = bio;
        this.work = work;
        this.education = education;
        this.travel = travel;
        this.interests = interests;
        this.hobbies = hobbies;
        this.website = website;
        this.linkedin = linkedin;
        this.instagram = instagram;
        this.twitter = twitter;
    }

    getBio() { return this.bio; }
    setBio(bio) { this.bio = bio; }

    getWork() { return this.work; }
    setWork(work) { this.work = work; }

    getEducation() { return this.education; }
    setEducation(education) { this.education = education; }

    getTravel() { return this.travel; }
    setTravel(travel) { this.travel = travel; }

    getInterests() { return this.interests; }
    setInterests(interests) { this.interests = interests; }

    getHobbies() { return this.hobbies; }
    setHobbies(hobbies) { this.hobbies = hobbies; }

    getWebsite() { return this.website; }
    setWebsite(website) { this.website = website; }

    getLinkedin() { return this.linkedin; }
    setLinkedin(linkedin) { this.linkedin = linkedin; }

    getInstagram() { return this.instagram; }
    setInstagram(instagram) { this.instagram = instagram; }

    getTwitter() { return this.twitter; }
    setTwitter(twitter) { this.twitter = twitter; }

    getAllData() {
        return {
            bio: this.bio,
            work: this.work,
            education: this.education,
            travel: this.travel,
            interests: this.interests,
            hobbies: this.hobbies,
            website: this.website,
            linkedin: this.linkedin,
            instagram: this.instagram,
            twitter: this.twitter
        };
    }
}