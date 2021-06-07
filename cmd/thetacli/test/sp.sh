#!/usr/bin/expect -f
set timeout -1
set arg1 [lindex $argv 0]
set arg2 [lindex $argv 1]
set arg3 [lindex $argv 2]
set arg4 [lindex $argv 3]
set arg5 [lindex $argv 4]
set arg6 [lindex $argv 5]
set arg7 [lindex $argv 6]
set arg8 [lindex $argv 7]
spawn thetacli tx service_payment --chain="privatenet" $arg1 $arg2 $arg3 $arg4 $arg5 $arg6 $arg7 $arg8
set code [open "./defaultpw" r]
set pass [read $code]
expect {
        password: {send "$pass\r" ; exp_continue}
        eof exit
}
