<script setup>
import { computed, onMounted, ref } from 'vue'
import {
  createGroupEvent,
  deleteGroupEvent,
  getGroupEvents,
  getInviteUsers,
  inviteUserToGroup,
  removeEventRSVP,
  setEventRSVP,
  undoGroupInvitation,
} from '@/api/groups/Groups.js'
import IconGlyph from '@/components/layout/IconGlyph.vue'

const props = defineProps({ groupId: { type: [String, Number], required: true } })
const events = ref([])
const title = ref('')
const description = ref('')
const startsAt = ref('')
const inviteUsers = ref([])
const inviteSearch = ref('')
const inviteBusyUserIDs = ref(new Set())
const eventBusy = ref(false)
const busyEventActions = ref({})
const message = ref('')
const messageIsError = ref(false)

function showMessage(text, isError = false) {
  message.value = text
  messageIsError.value = isError
}

const filteredInviteUsers = computed(() => {
  const search = inviteSearch.value.trim().toLowerCase()
  if (!search) return inviteUsers.value

  return inviteUsers.value.filter((user) => {
    const username = user.username?.toLowerCase() || ''
    const firstName = user.firstName?.toLowerCase() || ''
    const lastName = user.lastName?.toLowerCase() || ''
    const fullName = `${firstName} ${lastName}`.trim()
    return username.includes(search) || firstName.includes(search) || lastName.includes(search) || fullName.includes(search)
  })
})

async function load() {
  try {
    const [eventResult, inviteResult] = await Promise.all([
      getGroupEvents(props.groupId),
      getInviteUsers(props.groupId),
    ])
    events.value = eventResult?.events || []
    inviteUsers.value = inviteResult?.users || []
  } catch (error) {
    showMessage(error.message, true)
  }
}

async function createEvent() {
  if (eventBusy.value) return
  const when = new Date(startsAt.value)
  if (!title.value.trim() || !description.value.trim() || Number.isNaN(when.getTime())) {
    showMessage('Give the event a title, a description and a date and time.', true)
    return
  }
  if (when.getTime() <= Date.now()) {
    showMessage('The event has to be in the future.', true)
    return
  }
  eventBusy.value = true
  try {
    const result = await createGroupEvent(props.groupId, { title: title.value, description: description.value, startsAt: new Date(startsAt.value).toISOString() })
    title.value = ''; description.value = ''; startsAt.value = ''
    showMessage('Event created. Other members have been notified.')
    events.value = [...events.value, result.event].sort((a, b) => a.startsAt.localeCompare(b.startsAt) || a.id - b.id)
  } catch (error) { showMessage(error.message, true) } finally { eventBusy.value = false }
}

async function invite(user) {
  if (inviteBusyUserIDs.value.has(user.id)) return
  inviteBusyUserIDs.value = new Set(inviteBusyUserIDs.value).add(user.id)
  try {
    const result = await inviteUserToGroup(props.groupId, user.id)
    user.isInvited = true
    user.invitationId = result.invitationId
    showMessage('Invitation sent.')
  } catch (error) { showMessage(error.message, true) } finally {
    const busyUsers = new Set(inviteBusyUserIDs.value)
    busyUsers.delete(user.id)
    inviteBusyUserIDs.value = busyUsers
  }
}

async function undoInvitation(user) {
  if (inviteBusyUserIDs.value.has(user.id) || !user.invitationId) return
  inviteBusyUserIDs.value = new Set(inviteBusyUserIDs.value).add(user.id)
  try {
    await undoGroupInvitation(props.groupId, user.invitationId)
    user.isInvited = false
    user.invitationId = null
    showMessage('Invitation cancelled.')
  } catch (error) { showMessage(error.message, true) } finally {
    const busyUsers = new Set(inviteBusyUserIDs.value)
    busyUsers.delete(user.id)
    inviteBusyUserIDs.value = busyUsers
  }
}

onMounted(load)
async function respond(event, response) {
  if (busyEventActions.value[event.id]) return
  setEventBusy(event.id, 'rsvp')
  try {
    if (event.response === response) {
      const result = await removeEventRSVP(props.groupId, event.id)
      updateEventRSVP(event, '', { id: result.userId })
      showMessage('Response removed.')
    } else {
      const result = await setEventRSVP(event.id, response)
      updateEventRSVP(event, response, result.user)
      showMessage('Response saved.')
    }
  } catch (error) { showMessage(error.message, true) } finally { setEventBusy(event.id, '') }
}

async function deleteEvent(event) {
  if (busyEventActions.value[event.id] || !window.confirm('Delete this event?')) return
  setEventBusy(event.id, 'delete')
  try {
    await deleteGroupEvent(props.groupId, event.id)
    events.value = events.value.filter((item) => item.id !== event.id)
    showMessage('Event deleted.')
  } catch (error) { showMessage(error.message, true) } finally { setEventBusy(event.id, '') }
}

function updateEventRSVP(event, response, user) {
  event.goingUsers = (event.goingUsers || []).filter((voter) => voter.id !== user.id)
  event.notGoingUsers = (event.notGoingUsers || []).filter((voter) => voter.id !== user.id)
  if (response === 'going') event.goingUsers.push(user)
  if (response === 'declined') event.notGoingUsers.push(user)
  event.goingCount = event.goingUsers.length
  event.notGoingCount = event.notGoingUsers.length
  event.response = response
}

function setEventBusy(eventId, action) {
  const actions = { ...busyEventActions.value }
  if (action) actions[eventId] = action
  else delete actions[eventId]
  busyEventActions.value = actions
}

function voterName(user) {
  return `${user.firstName || ''} ${user.lastName || ''}`.trim() || user.username || 'Group member'
}
</script>

<template>
  <div class="activity">
    <div class="invite-block">
      <p
        v-if="message"
        class="activity-message"
        :class="{ 'activity-message--error': messageIsError }"
        :role="messageIsError ? 'alert' : 'status'"
      >{{ message }}</p>

      <section class="invite-panel orbit-surface">
      <header class="section-heading">
        <div>
          <p class="orbit-meta">Membership</p>
          <h2>Invite people</h2>
        </div>
        <span>{{ inviteUsers.length }} available</span>
      </header>
      <label for="invite-user">Search by username or name</label>
      <div class="search-field">
        <IconGlyph name="search" :size="17" />
        <input id="invite-user" v-model="inviteSearch" type="search" autocomplete="off" placeholder="Search by username or name..." />
      </div>
      <div v-if="filteredInviteUsers.length" class="invite-list" aria-label="People to invite">
        <div v-for="user in filteredInviteUsers" :key="user.id" class="invite-candidate">
          <span class="invite-avatar">
            <img v-if="user.avatarPath" :src="`/uploads/${user.avatarPath}`" alt="" />
            <span v-else>{{ `${user.firstName}${user.lastName}`.slice(0, 2).toUpperCase() }}</span>
          </span>
          <span class="invite-identity"><strong>{{ user.firstName }} {{ user.lastName }}</strong><small>@{{ user.username || 'orbit member' }}</small></span>
          <span v-if="user.isInvited" class="invite-actions">
            <span class="invite-status"><IconGlyph name="check" :size="14" /> Invited</span>
            <button class="button-secondary" type="button" :disabled="inviteBusyUserIDs.has(user.id)" @click="undoInvitation(user)">
              {{ inviteBusyUserIDs.has(user.id) ? 'Undoing...' : 'Undo' }}
            </button>
          </span>
          <button v-else class="button-primary" type="button" :disabled="inviteBusyUserIDs.has(user.id)" @click="invite(user)">
            {{ inviteBusyUserIDs.has(user.id) ? 'Sending...' : 'Invite' }}
          </button>
        </div>
      </div>
      <p v-else class="empty-state">No users found.</p>
      </section>
    </div>

    <section class="events-section">
      <header class="section-heading section-heading--major">
        <div>
          <p class="orbit-meta">Calendar</p>
          <h2>Events</h2>
        </div>
        <span>{{ events.length }} {{ events.length === 1 ? 'event' : 'events' }}</span>
      </header>

      <form class="event-planner orbit-surface" @submit.prevent="createEvent">
        <h3>Plan an event</h3>
        <div class="event-planner__grid">
          <label for="event-title">Title<input id="event-title" v-model="title" maxlength="50" required /></label>
          <label for="event-time">Date and time<input id="event-time" v-model="startsAt" type="datetime-local" required /></label>
          <label class="event-planner__description" for="event-description">Description<textarea id="event-description" v-model="description" maxlength="500" rows="3" required /></label>
        </div>
        <button class="button-primary" :disabled="eventBusy">{{ eventBusy ? 'Creating...' : 'Create event' }}</button>
      </form>

      <p v-if="!events.length" class="empty-state">No events yet.</p>
      <div v-else class="event-list">
        <article v-for="event in events" :key="event.id" class="event-card orbit-surface">
          <header class="event-heading">
            <div>
              <h3>{{ event.title }}</h3>
              <time><IconGlyph name="calendar" :size="15" /> {{ event.startsAt ? new Date(event.startsAt).toLocaleString() : 'Date not set' }}</time>
            </div>
            <button v-if="event.isCreator" type="button" class="button-danger" :disabled="Boolean(busyEventActions[event.id])" @click="deleteEvent(event)">
              {{ busyEventActions[event.id] === 'delete' ? 'Deleting...' : 'Delete event' }}
            </button>
          </header>
          <p class="event-description">{{ event.description }}</p>
          <div class="responses" aria-label="Your RSVP">
            <button type="button" :disabled="Boolean(busyEventActions[event.id])" :aria-pressed="event.response === 'going'" @click="respond(event, 'going')">
              <span class="response-check"><IconGlyph v-if="event.response === 'going'" name="check" :size="15" /></span>
              Going <strong>{{ event.goingCount || 0 }}</strong>
            </button>
            <button type="button" :disabled="Boolean(busyEventActions[event.id])" :aria-pressed="event.response === 'declined'" @click="respond(event, 'declined')">
              <span class="response-check"><IconGlyph v-if="event.response === 'declined'" name="check" :size="15" /></span>
              Not going <strong>{{ event.notGoingCount || 0 }}</strong>
            </button>
          </div>
          <details class="event-voters">
            <summary>View voters ({{ (event.goingCount || 0) + (event.notGoingCount || 0) }})</summary>
            <div class="event-voter-columns">
              <section>
                <h4>Going ({{ event.goingCount || 0 }})</h4>
                <p v-if="!event.goingUsers?.length" class="event-voters-empty">No responses yet.</p>
                <ul v-else>
                  <li v-for="user in event.goingUsers" :key="user.id">
                    <span class="event-voter-avatar"><img v-if="user.avatarPath" :src="`/uploads/${user.avatarPath}`" alt="" /><span v-else>{{ voterName(user).slice(0, 2).toUpperCase() }}</span></span>
                    <span><strong>{{ voterName(user) }}</strong><small v-if="user.username">@{{ user.username }}</small></span>
                  </li>
                </ul>
              </section>
              <section>
                <h4>Not going ({{ event.notGoingCount || 0 }})</h4>
                <p v-if="!event.notGoingUsers?.length" class="event-voters-empty">No responses yet.</p>
                <ul v-else>
                  <li v-for="user in event.notGoingUsers" :key="user.id">
                    <span class="event-voter-avatar"><img v-if="user.avatarPath" :src="`/uploads/${user.avatarPath}`" alt="" /><span v-else>{{ voterName(user).slice(0, 2).toUpperCase() }}</span></span>
                    <span><strong>{{ voterName(user) }}</strong><small v-if="user.username">@{{ user.username }}</small></span>
                  </li>
                </ul>
              </section>
            </div>
          </details>
        </article>
      </div>
    </section>
  </div>
</template>

<style scoped>
.activity { display: grid; gap: var(--space-7); }
.invite-block { display: grid; gap: var(--space-3); }
.activity-message { margin: 0; padding: var(--space-3) var(--space-4); border-left: 3px solid var(--color-mint); background: var(--color-surface-teal); color: var(--color-text-soft); }
.activity-message--error { border-left-color: var(--color-coral); background: rgb(var(--rgb-coral) / 8%); color: var(--color-coral); }
.invite-panel, .event-planner, .event-card { padding: var(--space-5); }
.section-heading { display: flex; align-items: end; justify-content: space-between; gap: var(--space-4); margin-bottom: var(--space-4); }
.section-heading--major { padding-top: var(--space-6); border-top: 1px solid var(--color-border); }
.section-heading h2 { margin: var(--space-1) 0 0; font-family: var(--font-display); font-size: 1.5rem; letter-spacing: 0; }
.section-heading--major h2 { font-size: 1.75rem; }
.section-heading .orbit-meta { margin: 0; color: var(--color-text-faint); }
.section-heading > span { flex: 0 0 auto; color: var(--color-text-muted); font-size: .8125rem; }
label { color: var(--color-text-muted); font-size: .875rem; }
input, textarea { min-width: 0; width: 100%; padding: var(--space-3); border: 1px solid var(--color-border); border-radius: var(--radius-small); background: var(--color-input); color: var(--color-text); }
textarea { line-height: 1.5; }
.search-field { display: flex; align-items: center; gap: var(--space-2); margin-top: var(--space-2); padding-left: var(--space-3); border: 1px solid var(--color-border); border-radius: var(--radius-small); background: var(--color-input); color: var(--color-text-faint); }
.search-field:focus-within { border-color: var(--color-violet); box-shadow: var(--focus-ring); }
.search-field input { padding-left: 0; border: 0; outline: 0; background: transparent; box-shadow: none; }
.invite-list { max-height: 18rem; margin-top: var(--space-4); overflow: auto; border-block: 1px solid var(--color-border); }
.invite-candidate { display: grid; grid-template-columns: 2.25rem minmax(0, 1fr) auto; align-items: center; gap: var(--space-3); min-height: 4rem; padding: var(--space-2); border-bottom: 1px solid var(--color-border); }
.invite-candidate:last-child { border-bottom: 0; }
.invite-identity { display: grid; min-width: 0; gap: var(--space-1); }
.invite-identity strong, .invite-identity small { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.invite-identity small, .invite-status { color: var(--color-text-muted); font-size: .75rem; }
.invite-avatar, .event-voter-avatar { display: grid; place-items: center; overflow: hidden; border-radius: 50%; background: var(--gradient-action); color: white; font-weight: 700; }
.invite-avatar { width: 2.25rem; height: 2.25rem; font-size: .7rem; }
.invite-avatar img, .event-voter-avatar img { width: 100%; height: 100%; object-fit: cover; }
.invite-actions, .invite-status { display: flex; align-items: center; gap: var(--space-2); }
.invite-status { color: var(--color-mint); }
.activity button { min-height: var(--touch-target); padding: 0 var(--space-4); border-radius: var(--radius-small); cursor: pointer; font-weight: 600; }
.button-primary { border: 0; background: var(--gradient-action); color: white; }
.button-secondary { border: 1px solid var(--color-border); background: transparent; color: var(--color-text-soft); }
.button-danger { border: 1px solid var(--color-coral); background: transparent; color: var(--color-coral); }
.button-secondary:hover:not(:disabled) { border-color: var(--color-violet); background: var(--color-input); }
.button-danger:hover:not(:disabled) { background: var(--color-coral); color: var(--color-background); }
.events-section { display: grid; gap: var(--space-5); }
.event-planner { display: grid; gap: var(--space-4); }
.event-planner h3 { margin: 0; font-size: 1.05rem; }
.event-planner__grid { display: grid; grid-template-columns: minmax(0, 1fr) minmax(14rem, .7fr); gap: var(--space-4); }
.event-planner__grid label { display: grid; gap: var(--space-2); }
.event-planner__description { grid-column: 1 / -1; }
.event-planner .button-primary { justify-self: end; }
.event-list { display: grid; gap: var(--space-4); }
.event-card { min-width: 0; }
.event-heading { display: flex; align-items: start; justify-content: space-between; gap: var(--space-4); }
.event-heading > div { min-width: 0; }
.event-heading h3 { margin: 0; font-family: var(--font-display); font-size: 1.2rem; letter-spacing: 0; }
.event-heading time { display: flex; align-items: center; gap: var(--space-2); margin-top: var(--space-2); color: var(--color-amber-soft); font-size: .8125rem; }
.event-description { margin: var(--space-4) 0; color: var(--color-text-soft); line-height: 1.6; white-space: pre-wrap; }
.responses { display: flex; flex-wrap: wrap; gap: var(--space-2); }
.responses button { display: inline-flex; min-width: 8.75rem; align-items: center; justify-content: center; gap: var(--space-2); border: 1px solid var(--color-border); background: var(--color-input); color: var(--color-text-muted); }
.responses button strong { color: var(--color-text); font-family: var(--font-meta); font-size: .75rem; }
.responses button[aria-pressed="true"] { border-color: var(--color-mint); background: var(--color-surface-teal); color: var(--color-mint-soft); box-shadow: inset 0 0 0 1px var(--color-mint); }
.response-check { display: grid; width: 1rem; height: 1rem; place-items: center; }
.event-voters { margin-top: var(--space-4); border-top: 1px solid var(--color-border); }
.event-voters summary { width: fit-content; padding-top: var(--space-3); color: var(--color-text-muted); cursor: pointer; font-size: .8125rem; font-weight: 600; }
.event-voter-columns { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: var(--space-5); padding-top: var(--space-4); }
.event-voters h4 { margin: 0 0 var(--space-3); font-size: .9rem; }
.event-voters ul { display: grid; gap: var(--space-2); margin: 0; padding: 0; list-style: none; }
.event-voters li { display: flex; min-width: 0; align-items: center; gap: var(--space-2); }
.event-voters li > span:last-child { display: grid; min-width: 0; }
.event-voters li strong, .event-voters li small { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.event-voters small, .event-voters-empty { margin: 0; color: var(--color-text-muted); font-size: .75rem; }
.event-voter-avatar { flex: 0 0 1.75rem; width: 1.75rem; height: 1.75rem; background: var(--color-input); font-size: .65rem; }
.empty-state { margin: 0; padding: var(--space-5); border: 1px dashed var(--color-border); border-radius: var(--radius-small); color: var(--color-text-muted); text-align: center; }
@media (max-width: 700px) {
  .activity { gap: var(--space-6); }
  .invite-panel, .event-planner, .event-card { padding: var(--space-4); }
  .section-heading--major { padding-top: var(--space-5); }
  .section-heading--major h2 { font-size: 1.5rem; }
  .event-planner__grid { grid-template-columns: 1fr; }
  .event-planner__description { grid-column: auto; }
}
@media (max-width: 520px) {
  .invite-candidate { grid-template-columns: 2.25rem minmax(0, 1fr); }
  .invite-candidate > .button-primary, .invite-actions { grid-column: 2; justify-self: start; }
  .event-heading { align-items: stretch; flex-direction: column; }
  .event-heading .button-danger, .event-planner .button-primary { width: 100%; justify-self: stretch; }
  .responses { display: grid; grid-template-columns: 1fr 1fr; }
  .responses button { min-width: 0; padding-inline: var(--space-2); }
  .event-voter-columns { grid-template-columns: 1fr; }
}
</style>
