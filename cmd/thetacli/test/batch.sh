#!/bin/bash
#set -x #echo on

Alice=2E833968E5bB786Ae419c4d13189fB081Cc43bab
Bob=70f587259738cB626A1720Af7038B8DcDb6a42a0
Carol=cd56123D0c5D6C1Ba4D39367b88cba61D93F5405
rid=rid1000001
rpduration=51

#####################################################################
# Default scale used by float functions.

float_scale=18
z18=1000000000000000000

#####################################################################
# Evaluate a floating point number expression.

function un_wei()
{
    local stat=0
    local result=0.0
    if [[ $# -gt 0 ]]; then
        result=$(echo "scale=$float_scale; $*/$z18" | bc -q 2>/dev/null)
        stat=$?
        if [[ $stat -eq 0  &&  -z "$result" ]]; then stat=1; fi
    fi
    echo $result
    return $stat
}


cbh=$(thetacli query status | tail -n +2 | jq .current_height)

aib=$(thetacli query account --address=$Alice | tail -n +2 | jq .coins.tfuelwei | tr -d '"')
aseq=$(thetacli query account --address=$Alice | tail -n +2 | jq .sequence | tr -d '"')
ans=$(( $aseq + 1 ))
echo "Alice initial balance: "$(un_wei $aib)
echo "Alice next sequence: "$ans
arf=$(thetacli query account --address=$Alice | tail -n +2 | jq .reserved_funds)

#echo "arf:"${#arf}

if [ ${#arf} == 2 ]; then
    echo "No Reserve Fund"
    echo "Alice begin create reserve."
    ./reserve.sh --from=$Alice --fund=10 --collateral=11 --duration=$rpduration --resource_ids=$rid --seq=$ans
    echo "Alice end create reserve."

    exit
else
    echo "Reserve Fund Exists"
    #echo "arf:"$arf
    ebh=$(echo "$arf" | jq .[0].end_block_height)
    echo "end_block_height:"$ebh
    echo "current_block_height:"$(thetacli query status | tail -n +2 | jq .current_height)
fi


#printf "Alice :%d\n" $aib

#./send.sh --from=$Alice --to=$Bob --theta=0 --tfuel=1 --seq=$ans

aab=$(thetacli query account --address=$Alice | tail -n +2 | jq .coins.tfuelwei | tr -d '"')
bab=$(thetacli query account --address=$Bob | tail -n +2 | jq .coins.tfuelwei | tr -d '"')

echo "Alice after balance: "$(un_wei $aab)
echo "Bob after balance: "$(un_wei $bab)

aseq=$(thetacli query account --address=$Alice | tail -n +2 | jq .sequence | tr -d '"')
ans=$(( $aseq + 1 ))

declare -a sigs

i=0
bobcnt=1
while [ $i -lt $bobcnt ]; do
    sigout=$(./sp.sh --from=$Alice --to=$Bob --payment_seq=$i --reserve_seq=$ans --resource_id=$rid --tfuel=2 | tail -n +8 | jq .source.signature | tr -d '"')
    sigs+=($sigout)
    let i++
done

carolcnt=$(( $bobcnt+1 ))
while [ $i -lt $carolcnt ]; do
    sigout=$(./sp.sh --from=$Alice --to=$Carol --payment_seq=$i --reserve_seq=$ans --resource_id=$rid --tfuel=2 | tail -n +8 | jq .source.signature | tr -d '"')
    sigs+=($sigout)
    let i++
done

for sig in "${sigs[@]}"
do
    echo "SIG:"$sig
done

i=0
bobcnt=1
while [ $i -lt $bobcnt ]; do
    sig=${sigs[$i]}
    echo "Bob"$i":"$sig
    ./sp.sh --from=$Alice --to=$Bob --payment_seq=$i --reserve_seq=$ans --resource_id=$rid --tfuel=2 --on_chain --src_sig=$sig
    let i++
done
while [ $i -lt $carolcnt ]; do
    #sigout=$(./sp.sh --from=$Alice --to=$Bob --payment_seq=$i --reserve_seq=$ans --resource_id=$rid --tfuel=2 | tail -n +8 | jq .source.signature)
    #sigs+=($sigout)
    echo "Carol"$i":"${sigs[$i]}
    let i++
done


