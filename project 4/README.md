• Giana Yang

• Instructions for building, running, and testing your program.
Open four terminals, three backend and one frontend.
for backend:
go install backend.go
backend --listen 8090 --backend :8091,:8092 
backend --listen 8091 --backend :8090,:8092
backend --listen 8092 --backend :8090,:8091


for frontend:
go install main.go
frontend --listen 8080 --backend :8090,:8091,:8092

• The state of your work. Did you complete the assignment? If not, what is missing? If your assignment
is incomplete or has known bugs, I prefer that students let me know, rather than let me discover these
deficiencies on my own.
Major bugs:
- when leader node is down and come back up, think itself as the leader and cause a deadlock.
- delays when nodes coming back up.
- when there is only one node, stop responding.
- I am aware that I should implement election within servers instead of having the frontend announce it. 



• I used the queue strategy we talked about in class.

• Document your program’s resilience. For each of the test cases described above,
1. indicate if your program passes it, to the best of your knowledge.
- With two replicas, working normally.
- With one or two replicas down, working correctly but delay on updating the frontend. 
- When replicas come back from termination, working normally.
- With propser/learner down, takes more than three seconds to have a new leader working properly.

2. describe specifically how your program handles the situation. What messages are passed? What
can go wrong?
- When a replica is down, leader update the quorum number.
- When leader notice a node coming back up, update the quorum number, send the node all its log before continuing executions.
- When leader is down, client notice first and choose a new leader. 



• Explain any important design decisions you made in completing this assignment. In particular:
1. What replication strategy do you choose? Why did you choose it? How does it suit your application?
I chose multi-paxos. I chose it because it was a shopping list, I just need it to be correct with backend, time is not really
a rush but the correctness should be consistent. Instead of a leader election when the 
leader can just be the number with the least serial number. 

2. Weigh the pros and cons of your replication strategy.
pros:
A leader can get ahead with commands while replicas catch up later. 

cons:
Extremely slow because of the 2 phase commit. 

3. How does your application avoid blocking while waiting to receive messages?
Divide messages to different queue using their port number (unique node serial number)
Send each queue to a thread so leader would not wait for its own reply. 
