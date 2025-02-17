package raft

//
// this is an outline of the API that raft must expose to
// the service (or tester). see comments below for
// each of these functions for more details.
//
// rf = Make(...)
//   create a new Raft server.
// rf.Start(command interface{}) (index, term, isleader)
//   start agreement on a new log entry
// rf.GetState() (term, isLeader)
//   ask a Raft for its current term, and whether it thinks it is leader
// ApplyMsg
//   each time a new entry is committed to the log, each Raft peer
//   should send an ApplyMsg to the service (or tester)
//   in the same server.
//

import (
	"math/rand"
	//	"bytes"
	"sync"
	"sync/atomic"
	"time"

	//	"6.824/labgob"
	"a6.824/labrpc"
)

type ServerState int

const (
	Leader = iota
	Candidate
	Follower
)

//
// as each Raft peer becomes aware that successive log entries are
// committed, the peer should send an ApplyMsg to the service (or
// tester) on the same server, via the applyCh passed to Make(). set
// CommandValid to true to indicate that the ApplyMsg contains a newly
// committed log entry.
//
// in part 2D you'll want to send other kinds of messages (e.g.,
// snapshots) on the applyCh, but set CommandValid to false for these
// other uses.
//
type ApplyMsg struct {
	CommandValid bool
	Command      interface{}
	CommandIndex int

	// For 2D:
	SnapshotValid bool
	Snapshot      []byte
	SnapshotTerm  int
	SnapshotIndex int
}

//
// A Go object implementing a single Raft peer.
//
type Raft struct {
	mu        sync.Mutex          // Lock to protect shared access to this peer's state
	peers     []*labrpc.ClientEnd // RPC end points of all peers
	persister *Persister          // Object to hold this peer's persisted state
	me        int                 // this peer's index into peers[]
	dead      int32               // set by Kill()

	// Your data here (2A, 2B, 2C).
	// Look at the paper's Figure 2 for a description of what
	// state a Raft server must maintain.
	state       ServerState
	currenTerm  int
	voteFor     int
	log         []*LogEntry
	commitIndex int
	lastApplied int

	nextIndex  []int
	matchIndex []int

	heartbeatTimer *time.Timer
	electionTimer  *time.Timer

	applyCh   chan ApplyMsg
	applyCond *sync.Cond
}

type LogEntry struct {
	Command interface{}
	Term    int
}

type AppendEntries struct {
	Term          int
	LearId        int
	PrevLogIndex  int
	PrevTermIndex int
	Entries       []*LogEntry

	LeaderCommitId int
}

// return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	//var term int
	//var isleader bool
	// Your code here (2A).
	return rf.currenTerm, rf.state == Leader
}

//
// save Raft's persistent state to stable storage,
// where it can later be retrieved after a crash and restart.
// see paper's Figure 2 for a description of what should be persistent.
//
func (rf *Raft) persist() {
	// Your code here (2C).
	// Example:
	// w := new(bytes.Buffer)
	// e := labgob.NewEncoder(w)
	// e.Encode(rf.xxx)
	// e.Encode(rf.yyy)
	// data := w.Bytes()
	// rf.persister.SaveRaftState(data)
}

//
// restore previously persisted state.
//
func (rf *Raft) readPersist(data []byte) {
	if data == nil || len(data) < 1 { // bootstrap without any state?
		return
	}
	// Your code here (2C).
	// Example:
	// r := bytes.NewBuffer(data)
	// d := labgob.NewDecoder(r)
	// var xxx
	// var yyy
	// if d.Decode(&xxx) != nil ||
	//    d.Decode(&yyy) != nil {
	//   error...
	// } else {
	//   rf.xxx = xxx
	//   rf.yyy = yyy
	// }
}

//
// A service wants to switch to snapshot.  Only do so if Raft hasn't
// have more recent info since it communicate the snapshot on applyCh.
//
func (rf *Raft) CondInstallSnapshot(lastIncludedTerm int, lastIncludedIndex int, snapshot []byte) bool {

	// Your code here (2D).

	return true
}

// the service says it has created a snapshot that has
// all info up to and including index. this means the
// service no longer needs the log through (and including)
// that index. Raft should now trim its log as much as possible.
func (rf *Raft) Snapshot(index int, snapshot []byte) {
	// Your code here (2D).

}

//
// example RequestVote RPC arguments structure.
// field names must start with capital letters!
//
type RequestVoteArgs struct {
	// Your data here (2A, 2B).
	Term          int
	CandidateId   int
	LastLogIndex  int
	LastTermIndex int
}

//
// example RequestVote RPC reply structure.
// field names must start with capital letters!
//
type RequestVoteReply struct {
	// Your data here (2A).
	Term        int
	VoteGranted bool
}

//
// example RequestVote RPC handler.
//
func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	// Your code here (2A, 2B).
	rf.mu.Lock()
	defer rf.mu.Unlock()
	//DPrintf("server %v get a requestvote from server %v", rf.me, args.CandidateId)
	DPrintf("{Node %v}'s state is {state %v,term %v,voteFor %v} "+
		"before processing requestVoteRequest %v and reply requestVoteResponse %v", rf.me, rf.state,
		rf.currenTerm, rf.voteFor, args, reply)
	//当follower的term更大或者 term相等但是在给term内已经投出Leader票了，此时投出拒绝票
	if args.Term < rf.currenTerm || (args.Term == rf.currenTerm && rf.voteFor != -1 && rf.voteFor != args.CandidateId) {
		reply.Term, reply.VoteGranted = rf.currenTerm, false
		DPrintf("server %v reject the requestvote from server %v", rf.me, args.CandidateId)
		return
	}
	rf.state, rf.voteFor, rf.currenTerm = Follower, args.CandidateId, args.Term
	rf.electionTimer.Reset(makeRandomElectionTimeout())
	reply.Term, reply.VoteGranted = args.Term, true
	DPrintf("server %v pass the requestvote from server %v", rf.me, args.CandidateId)
}

//
// example code to send a RequestVote RPC to a server.
// server is the index of the target server in rf.peers[].
// expects RPC arguments in args.
// fills in *reply with RPC reply, so caller should
// pass &reply.
// the types of the args and reply passed to Call() must be
// the same as the types of the arguments declared in the
// handler function (including whether they are pointers).
//
// The labrpc package simulates a lossy network, in which servers
// may be unreachable, and in which requests and replies may be lost.
// Call() sends a request and waits for a reply. If a reply arrives
// within a timeout interval, Call() returns true; otherwise
// Call() returns false. Thus Call() may not return for a while.
// A false return can be caused by a dead server, a live server that
// can't be reached, a lost request, or a lost reply.
//
// Call() is guaranteed to return (perhaps after a delay) *except* if the
// handler function on the server side does not return.  Thus there
// is no need to implement your own timeouts around Call().
//
// look at the comments in ../labrpc/labrpc.go for more details.
//
// if you're having trouble getting RPC to work, check that you've
// capitalized all field names in structs passed over RPC, and
// that the caller passes the address of the reply struct with &, not
// the struct itself.
//
func (rf *Raft) sendRequestVote(server int, args *RequestVoteArgs, reply *RequestVoteReply) bool {
	ok := rf.peers[server].Call("Raft.RequestVote", args, reply)
	return ok
}

//
// the service using Raft (e.g. a k/v server) wants to start
// agreement on the next command to be appended to Raft's log. if this
// server isn't the leader, returns false. otherwise start the
// agreement and return immediately. there is no guarantee that this
// command will ever be committed to the Raft log, since the leader
// may fail or lose an election. even if the Raft instance has been killed,
// this function should return gracefully.
//
// the first return value is the index that the command will appear at
// if it's ever committed. the second return value is the current
// term. the third return value is true if this server believes it is
// the leader.
//
func (rf *Raft) Start(command interface{}) (int, int, bool) {
	index := -1
	term := -1
	isLeader := true

	// Your code here (2B).

	return index, term, isLeader
}

//
// the tester doesn't halt goroutines created by Raft after each test,
// but it does call the Kill() method. your code can use killed() to
// check whether Kill() has been called. the use of atomic avoids the
// need for a lock.
//
// the issue is that long-running goroutines use memory and may chew
// up CPU time, perhaps causing later tests to fail and generating
// confusing debug output. any goroutine with a long-running loop
// should call killed() to check whether it should stop.
//
func (rf *Raft) Kill() {
	atomic.StoreInt32(&rf.dead, 1)
	// Your code here, if desired.
}

func (rf *Raft) killed() bool {
	z := atomic.LoadInt32(&rf.dead)
	return z == 1
}

// The ticker go routine starts a new election if this peer hasn't received
// heartsbeats recently.
func (rf *Raft) ticker() {
	for rf.killed() == false {
		// Your code here to check if a leader election should
		// be started and to randomize sleeping time using
		// time.Sleep().
		select {
		case <-rf.heartbeatTimer.C:
			//DPrintf("node %v is state %v in term %v", rf.me, rf.state, rf.currenTerm)
			rf.mu.Lock()
			if rf.state == Leader {
				DPrintf("node %v is Leader in term %v", rf.me, rf.currenTerm)
				rf.broadcastHeartbeat()
			}
			rf.heartbeatTimer.Reset(makeStableHeartbeatTimeout())
			rf.mu.Unlock()
		case <-rf.electionTimer.C:
			DPrintf("server %v electionTimer timeout", rf.me)
			rf.mu.Lock()
			if rf.state != Leader {
				rf.state = Candidate
				rf.currenTerm++
				rf.startElection()
			}
			rf.electionTimer.Reset(makeRandomElectionTimeout())
			rf.mu.Unlock()
		}
	}
}

func (rf *Raft) startElection() {
	DPrintf("server %v start a election", rf.me)
	grantedVote := 1
	rf.voteFor = rf.me
	args := RequestVoteArgs{
		Term:          rf.currenTerm,
		CandidateId:   rf.me,
		LastLogIndex:  rf.commitIndex,
		LastTermIndex: rf.currenTerm - 1,
	}
	for peer := range rf.peers {
		if peer == rf.me {
			continue
		}
		go func(peer int) {
			reply := new(RequestVoteReply)
			DPrintf("server %v send a requestvote to server %v", rf.me, peer)
			rf.sendRequestVote(peer, &args, reply)
			DPrintf("server %v get a reply from server %v", rf.me, reply)
			rf.mu.Lock()
			defer rf.mu.Unlock()
			if rf.currenTerm == reply.Term && rf.state == Candidate {
				if reply.VoteGranted {
					grantedVote++
					if grantedVote > len(rf.peers)/2 {
						DPrintf("server %v get major votes %v / %v", rf.me, grantedVote, len(rf.peers))
						rf.state = Leader
						rf.broadcastHeartbeat()
					} else {
						DPrintf("server %v get minor votes %v / %v", rf.me, grantedVote, len(rf.peers))
					}
				} else if reply.Term > rf.currenTerm {
					rf.state = Follower
					rf.currenTerm = reply.Term
					rf.voteFor = -1
					rf.electionTimer.Reset(makeRandomElectionTimeout())
				}
			}
		}(peer)
	}
}

func (rf *Raft) broadcastHeartbeat() {
	for _, peer := range rf.peers {
		args := AppendEntriesArgs{}
		reply := AppendEntriesReply{}
		peer.Call("Raft.AppendEntries", &args, &reply)
	}
}

type AppendEntriesArgs struct {
	Entry AppendEntries
	Term  int
}

type AppendEntriesReply struct {
}

func (rf *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) {
	/*if args.Term > rf.currenTerm {
		rf.state = Follower
		rf.voteFor = -1
		rf.currenTerm = args.Term
	}*/
	rf.electionTimer.Reset(makeRandomElectionTimeout())
}

func makeStableHeartbeatTimeout() (d time.Duration) {
	return time.Millisecond * 150
}

func makeRandomElectionTimeout() time.Duration {
	i := rand.Intn(300) + 1200
	return time.Duration(i) * time.Millisecond
}

//
// the service or tester wants to create a Raft server. the ports
// of all the Raft servers (including this one) are in peers[]. this
// server's port is peers[me]. all the servers' peers[] arrays
// have the same order. persister is a place for this server to
// save its persistent state, and also initially holds the most
// recent saved state, if any. applyCh is a channel on which the
// tester or service expects Raft to send ApplyMsg messages.
// Make() must return quickly, so it should start goroutines
// for any long-running work.
//
func Make(peers []*labrpc.ClientEnd, me int,
	persister *Persister, applyCh chan ApplyMsg) *Raft {
	rf := Raft{
		peers:          peers,
		persister:      persister,
		me:             me,
		dead:           0,
		state:          Follower,
		currenTerm:     0,
		voteFor:        -1,
		log:            make([]*LogEntry, 1),
		commitIndex:    0,
		lastApplied:    0,
		nextIndex:      make([]int, len(peers)),
		matchIndex:     make([]int, len(peers)),
		heartbeatTimer: time.NewTimer(makeStableHeartbeatTimeout()),
		electionTimer:  time.NewTimer(makeRandomElectionTimeout()),
		applyCh:        applyCh,
	}

	// Your initialization code here (2A, 2B, 2C).

	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())
	rf.applyCond = sync.NewCond(&rf.mu)
	for i := 0; i < len(peers); i++ {
		rf.matchIndex[i], rf.nextIndex[i] = 0, rf.commitIndex+1
		//if i != rf.me {
		//	rf.replicatorCond[i] = sync.NewCond(&sync.Mutex{})
		//	// start replicator goroutine to replicate entries in batch
		//	go rf.replicator(i)
		//}
	}
	// start ticker goroutine to start elections
	go rf.ticker()

	return &rf
}
