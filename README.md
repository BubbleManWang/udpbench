# Iteration 1 - memory issues

This iteration takes the usual game loop method of having a main loop where we call Update every 'tick'. Update handles the recv'd packets, sends pings if necessary, then checks if clients have timed out or not.

The NetcodeConn object has a single go routine that handles the read calls, it then sends the read results to a channel which takes the data and calls a bound delegate function that is set by the caller, Server in this case.

Unfortunately performance is pretty garbage. We can't even service up to 2000 clients. Half don't get any responses, and the ones that do only get about 20 out of 301 iterations.

At 1000 clients we get 74-90 packets recv'd by clients out of the 301 iterations... this needs to be improved.

Memory usage is also pretty terrible, the process was up to about 500MB for 2000 clients, and continues to climb on every run of 2000 more clients.
`
C:\gohome\src\github.com\wirepair\udpbench>go tool pprof udpbench.exe mem.pprof
Entering interactive mode (type "help" for commands)
(pprof) top10
401.92MB of 402.72MB total (99.80%)
Dropped 60 nodes (cum <= 2.01MB)
Showing top 10 nodes out of 12 (cum >= 18.07MB)
      flat  flat%   sum%        cum   cum%
  362.78MB 90.08% 90.08%   394.06MB 97.85%  main.(*NetcodeConn).read
   18.07MB  4.49% 94.57%    18.07MB  4.49%  syscall.(*RawSockaddrAny).Sockaddr
   13.20MB  3.28% 97.85%    31.27MB  7.77%  net.(*UDPConn).readFrom
    7.81MB  1.94% 99.79%     7.81MB  1.94%  runtime.makechan
    0.06MB 0.014% 99.80%     7.87MB  1.95%  main.NewServer
         0     0% 99.80%   394.06MB 97.85%  main.(*NetcodeConn).receiver
         0     0% 99.80%     7.87MB  1.95%  main.main
         0     0% 99.80%     7.87MB  1.95%  main.serveLoop
         0     0% 99.80%    31.27MB  7.77%  net.(*UDPConn).ReadFromUDP
         0     0% 99.80%    18.07MB  4.49%  net.(*netFD).readFrom
(pprof)
`

yikes, that read looks Bad, we need to improve this.

