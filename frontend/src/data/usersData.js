import { reactive } from 'vue';

export const userAbout = {
    Work: '',
    Education: '',
    Travel: '',
    Intrests: '',
    Hobbies: '',
    Website: '',
    Linkedin: '',
    Instgram: '',
    Twitter: ''
};

// two separate stores on purpose:
// profileData is ME (my profile page, edit profile, top bar avatar)
// viewedProfile is the OTHER person whose profile i am looking at.
// when both used one store, my own data loaded by the top bar could
// overwrite the profile i opened, so i saw myself "following myself"
function emptyProfile() {
    return {
        userInfo: {
            id: null,
            firstName: '',
            lastName: '',
            userName: '',
            email: '',
            about: '',
            dob: '',
            password: '',
            avatar: '',
            isPrivate: 0
        },

        numOfFollowers: 0,
        numOfFollowing: 0,
        numOfPosts: 0,

        about: {
            bio: '',
            work: '',
            education: '',
            travel: '',
            intrests: '',
            hobbies: '',
            website: '',
            linkedin: '',
            instgram: '',
            twitter: ''
        },

        followers: {},
        friends: {},
        following: {},
        isFollowing: -1,
        canMessage: false,
        show: false,
        isPrivate: 0
    }
}

export const profileData = reactive(emptyProfile());

export const viewedProfile = reactive(emptyProfile());
