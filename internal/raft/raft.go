// Package raft implements leader election, replicated logs, commit rules and persistence.
// Membership changes and log snapshots are intentionally outside this prototype's scope.
package raft

import (
 "errors"
 "fmt"
 "hash/fnv"
 "math/rand"
 "sync"
 "time"
)
type Role int
const(Follower Role=iota;Candidate;Leader)
func(r Role)String()string{switch r{case Leader:return "leader";case Candidate:return "candidate"};return "follower"}
type LogEntry struct{Term uint64 `json:"term"`;Command []byte `json:"command"`}
type RequestVoteArgs struct{Term uint64 `json:"term"`;CandidateID string `json:"candidate_id"`;LastLogIndex uint64 `json:"last_log_index"`;LastLogTerm uint64 `json:"last_log_term"`}
type RequestVoteReply struct{Term uint64 `json:"term"`;VoteGranted bool `json:"vote_granted"`}
type AppendEntriesArgs struct{Term uint64 `json:"term"`;LeaderID string `json:"leader_id"`;PrevLogIndex uint64 `json:"prev_log_index"`;PrevLogTerm uint64 `json:"prev_log_term"`;Entries []LogEntry `json:"entries"`;LeaderCommit uint64 `json:"leader_commit"`}
type AppendEntriesReply struct{Term uint64 `json:"term"`;Success bool `json:"success"`;ConflictIndex uint64 `json:"conflict_index"`}
type NotLeaderError struct{LeaderID string}
func(e *NotLeaderError)Error()string{return fmt.Sprintf("raft: not the leader (leader hint %q)",e.LeaderID)}
var(ErrTimeout=errors.New("raft: proposal timed out, outcome unknown");ErrLeadershipLost=errors.New("raft: leadership lost before the entry committed");ErrStopped=errors.New("raft: node stopped"))
type Transport interface{Call(peer,method string,args,reply any)error}
type Config struct{ID string;Peers []string;Transport Transport;DataDir string;Apply func(index uint64,command []byte)any;ElectionMin,ElectionMax,Heartbeat time.Duration}
type Status struct{ID string `json:"id"`;Role string `json:"role"`;Term uint64 `json:"term"`;LeaderID string `json:"leader_id"`;CommitIndex uint64 `json:"commit_index"`;LastApplied uint64 `json:"last_applied"`;LastIndex uint64 `json:"last_index"`;LeaderChanges uint64 `json:"leader_changes"`}
type result struct{val any;err error};type pending struct{term uint64;ch chan result}
const maxBatch=256
type Node struct{cfg Config;mu sync.Mutex;st *storage;cond *sync.Cond;term uint64;votedFor string;log []LogEntry;commit,applied uint64;role Role;leader string;next,match map[string]uint64;inflight map[string]bool;pend map[uint64]*pending;deadline,lastBeat time.Time;stopped bool;leaderChanges uint64;rnd *rand.Rand}
func New(cfg Config)(*Node,error){if cfg.ElectionMin<=0{cfg.ElectionMin=300*time.Millisecond};if cfg.ElectionMax<=cfg.ElectionMin{cfg.ElectionMax=2*cfg.ElectionMin};if cfg.Heartbeat<=0{cfg.Heartbeat=75*time.Millisecond};h:=fnv.New64a();_,_=h.Write([]byte(cfg.ID));n:=&Node{cfg:cfg,log:[]LogEntry{{}},next:map[string]uint64{},match:map[string]uint64{},inflight:map[string]bool{},pend:map[uint64]*pending{},rnd:rand.New(rand.NewSource(time.Now().UnixNano()^int64(h.Sum64())))};n.cond=sync.NewCond(&n.mu);if cfg.DataDir!=""{st,term,vote,entries,e:=openStorage(cfg.DataDir);if e!=nil{return nil,e};n.st,n.term,n.votedFor=st,term,vote,n.votedFor; n.votedFor=vote;n.log=append(n.log,entries...)};n.resetDeadlineLocked();go n.ticker();go n.applier();return n,nil}
func(n *Node)size()int{return len(n.cfg.Peers)+1}
func(n *Node)lastIndex()uint64{return uint64(len(n.log)-1)}
func(n *Node)must(e error){if e!=nil&&!n.stopped{panic(fmt.Sprintf("raft: fatal persistence failure: %v",e))}}
func(n *Node)persistMeta(){if n.st!=nil{n.must(n.st.saveMeta(n.term,n.votedFor))}}
func(n *Node)resetDeadlineLocked(){span:=n.cfg.ElectionMax-n.cfg.ElectionMin;n.deadline=time.Now().Add(n.cfg.ElectionMin+time.Duration(n.rnd.Int63n(int64(span)+1)))}
func(n *Node)stepDownLocked(term uint64){if term>n.term{n.term=term;n.votedFor="";n.leader="";n.persistMeta()};n.role=Follower}
func(n *Node)appendLocked(e LogEntry)uint64{n.log=append(n.log,e);if n.st!=nil{n.must(n.st.append([]LogEntry{e}))};return n.lastIndex()}
func(n *Node)ticker(){t:=time.NewTicker(10*time.Millisecond);defer t.Stop();for range t.C{n.mu.Lock();if n.stopped{n.mu.Unlock();return};now:=time.Now();if n.role==Leader{if now.Sub(n.lastBeat)>=n.cfg.Heartbeat{n.broadcastLocked()}}else if now.After(n.deadline){n.startElectionLocked()};n.mu.Unlock()}}
func(n *Node)startElectionLocked(){n.role=Candidate;n.term++;n.votedFor=n.cfg.ID;n.leader="";n.persistMeta();n.resetDeadlineLocked();term:=n.term;last:=n.lastIndex();args:=&RequestVoteArgs{Term:term,CandidateID:n.cfg.ID,LastLogIndex:last,LastLogTerm:n.log[last].Term};votes:=1;if votes*2>n.size(){n.becomeLeaderLocked();return};for _,peer:=range n.cfg.Peers{go func(p string){var reply RequestVoteReply;if n.cfg.Transport==nil||n.cfg.Transport.Call(p,"RequestVote",args,&reply)!=nil{return};n.mu.Lock();defer n.mu.Unlock();if n.stopped||n.role!=Candidate||n.term!=term{return};if reply.Term>n.term{n.stepDownLocked(reply.Term);return};if reply.VoteGranted{votes++;if votes*2>n.size(){n.becomeLeaderLocked()}}}(peer)}}
func(n *Node)becomeLeaderLocked(){n.role=Leader;n.leader=n.cfg.ID;n.leaderChanges++;for _,p:=range n.cfg.Peers{n.next[p]=n.lastIndex()+1;n.match[p]=0};n.appendLocked(LogEntry{Term:n.term});n.advanceCommitLocked();n.broadcastLocked()}
func(n *Node)broadcastLocked(){n.lastBeat=time.Now();for _,p:=range n.cfg.Peers{go n.replicateLoop(p,n.term)}}
func(n *Node)replicateLoop(peer string,term uint64){n.mu.Lock();if n.inflight[peer]{n.mu.Unlock();return};n.inflight[peer]=true;n.mu.Unlock();defer func(){n.mu.Lock();n.inflight[peer]=false;n.mu.Unlock()}();for n.replicateOnce(peer,term){}}
func(n *Node)replicateOnce(peer string,term uint64)bool{n.mu.Lock();if n.stopped||n.role!=Leader||n.term!=term{n.mu.Unlock();return false};last:=n.lastIndex();next:=n.next[peer];if next<1{next=1};if next>last+1{next=last+1};prev:=next-1;end:=last;if end-prev>maxBatch{end=prev+maxBatch};entries:=append([]LogEntry(nil),n.log[next:end+1]...);args:=&AppendEntriesArgs{Term:term,LeaderID:n.cfg.ID,PrevLogIndex:prev,PrevLogTerm:n.log[prev].Term,Entries:entries,LeaderCommit:n.commit};n.mu.Unlock();var reply AppendEntriesReply;if n.cfg.Transport==nil||n.cfg.Transport.Call(peer,"AppendEntries",args,&reply)!=nil{return false};n.mu.Lock();defer n.mu.Unlock();if n.stopped||n.role!=Leader||n.term!=term{return false};if reply.Term>n.term{n.stepDownLocked(reply.Term);return false};if reply.Success{m:=prev+uint64(len(entries));if m>n.match[peer]{n.match[peer]=m};n.next[peer]=n.match[peer]+1;n.advanceCommitLocked();return n.next[peer]<=n.lastIndex()};if next<=1{return false};if reply.ConflictIndex>=1&&reply.ConflictIndex<next{n.next[peer]=reply.ConflictIndex}else{n.next[peer]=next-1};return true}
func(n *Node)advanceCommitLocked(){for idx:=n.lastIndex();idx>n.commit;idx--{if n.log[idx].Term!=n.term{break};count:=1;for _,p:=range n.cfg.Peers{if n.match[p]>=idx{count++}};if count*2>n.size(){n.commit=idx;n.cond.Broadcast();return}}}
func(n *Node)HandleRequestVote(a *RequestVoteArgs)RequestVoteReply{n.mu.Lock();defer n.mu.Unlock();if n.stopped||a.Term<n.term{return RequestVoteReply{Term:n.term}};if a.Term>n.term{n.stepDownLocked(a.Term)};last:=n.lastIndex();lastTerm:=n.log[last].Term;up:=a.LastLogTerm>lastTerm||(a.LastLogTerm==lastTerm&&a.LastLogIndex>=last);if(n.votedFor==""||n.votedFor==a.CandidateID)&&up{n.votedFor=a.CandidateID;n.persistMeta();n.resetDeadlineLocked();return RequestVoteReply{Term:n.term,VoteGranted:true}};return RequestVoteReply{Term:n.term}}
func(n *Node)HandleAppendEntries(a *AppendEntriesArgs)AppendEntriesReply{n.mu.Lock();defer n.mu.Unlock();if n.stopped||a.Term<n.term{return AppendEntriesReply{Term:n.term}};if a.Term>n.term{n.stepDownLocked(a.Term)}else if n.role!=Follower{n.role=Follower};n.leader=a.LeaderID;n.resetDeadlineLocked();last:=n.lastIndex();if a.PrevLogIndex>last{return AppendEntriesReply{Term:n.term,ConflictIndex:last+1}};if n.log[a.PrevLogIndex].Term!=a.PrevLogTerm{ct:=n.log[a.PrevLogIndex].Term;ci:=a.PrevLogIndex;for ci>1&&n.log[ci-1].Term==ct{ci--};return AppendEntriesReply{Term:n.term,ConflictIndex:ci}};truncated:=false;var fresh []LogEntry;for i,e:=range a.Entries{idx:=a.PrevLogIndex+1+uint64(i);if idx<=n.lastIndex(){if n.log[idx].Term==e.Term{continue};n.log=n.log[:idx];truncated=true};n.log=append(n.log,e);fresh=append(fresh,e)};if n.st!=nil{if truncated{n.must(n.st.rewrite(n.log[1:]))}else if len(fresh)>0{n.must(n.st.append(fresh))}};if a.LeaderCommit>n.commit{c:=a.LeaderCommit;if max:=a.PrevLogIndex+uint64(len(a.Entries));c>max{c=max};if c>n.commit{n.commit=c;n.cond.Broadcast()}};return AppendEntriesReply{Term:n.term,Success:true}}
func(n *Node)applier(){for{n.mu.Lock();for n.applied>=n.commit&&!n.stopped{n.cond.Wait()};if n.stopped{n.mu.Unlock();return};idx:=n.applied+1;e:=n.log[idx];n.applied=idx;p:=n.pend[idx];delete(n.pend,idx);n.mu.Unlock();var v any;if len(e.Command)>0&&n.cfg.Apply!=nil{v=n.cfg.Apply(idx,e.Command)};if p!=nil{if p.term==e.Term{p.ch<-result{val:v}}else{p.ch<-result{err:ErrLeadershipLost}}}}}
func(n *Node)Propose(cmd []byte,timeout time.Duration)(any,error){n.mu.Lock();if n.stopped{n.mu.Unlock();return nil,ErrStopped};if n.role!=Leader{hint:=n.leader;n.mu.Unlock();return nil,&NotLeaderError{LeaderID:hint}};idx:=n.appendLocked(LogEntry{Term:n.term,Command:cmd});p:=&pending{term:n.term,ch:make(chan result,1)};n.pend[idx]=p;n.advanceCommitLocked();n.broadcastLocked();n.mu.Unlock();timer:=time.NewTimer(timeout);defer timer.Stop();select{case r:=<-p.ch:return r.val,r.err;case <-timer.C:n.mu.Lock();delete(n.pend,idx);n.mu.Unlock();return nil,ErrTimeout}}
func(n *Node)Status()Status{n.mu.Lock();defer n.mu.Unlock();return Status{ID:n.cfg.ID,Role:n.role.String(),Term:n.term,LeaderID:n.leader,CommitIndex:n.commit,LastApplied:n.applied,LastIndex:n.lastIndex(),LeaderChanges:n.leaderChanges}}
func(n *Node)Stop(){n.mu.Lock();defer n.mu.Unlock();if n.stopped{return};n.stopped=true;for i,p:=range n.pend{p.ch<-result{err:ErrStopped};delete(n.pend,i)};n.cond.Broadcast();if n.st!=nil{n.st.close()}}
