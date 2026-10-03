import { forgetSession } from '@/router/router'

// this function check the respond status after fetching and return false for unanothorized
export function checkSessionResponse(resp) {
    if(resp.status === 401 ) {
        // the cookie is gone or expired, the router must not trust the old answer
        forgetSession()
        return false;
    }
    return true
}
