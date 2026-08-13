## Test Results

### Cost tests performed at Thu Jun 10 13:17:06 EDT 2021
```
tfuel=100.0
Thu Jun 10 13:19:07 EDT 2021
TFuel(USD): 0.54810887040421
USD/Transaction: .16
100.0 TFuel in USD = 54.8109 x 10 items = 548.1090 USD
visacost USD: 7.57 to send 10 items (1.3800%)margin
mn30cost USD: 1.60 to send 10 items (.2900%)margin
```
```
tfuel=10.0
Thu Jun 10 13:19:40 EDT 2021
TFuel(USD): 0.54810887040421
USD/Transaction: .16
10.0 TFuel in USD = 5.4811 x 10 items = 54.8110 USD
visacost USD: 1.21 to send 10 items (2.2000%)margin
mn30cost USD: 1.60 to send 10 items (2.9100%)margin
```
```
tfuel=1.0
Thu Jun 10 13:20:36 EDT 2021
TFuel(USD): 0.54810887040421
USD/Transaction: .16
1.0 TFuel in USD = .5481 x 10 items = 5.4810 USD
visacost USD: .57 to send 10 items (10.3900%)margin
mn30cost USD: 1.60 to send 10 items (29.1900%)margin
```
```
tfuel=0.1
Thu Jun 10 13:21:20 EDT 2021
TFuel(USD): 0.54810887040421
USD/Transaction: .16
0.1 TFuel in USD = .0548 x 10 items = .5480 USD
visacost USD: .51 to send 10 items (93.0600%)margin
mn30cost USD: 1.60 to send 10 items (291.9700%)margin
```
```
tfuel=0.01
Thu Jun 10 13:22:17 EDT 2021
TFuel(USD): 0.55011933917353
USD/Transaction: .17
0.01 TFuel in USD = .0055 x 10 items = .0550 USD
visacost USD: .50 to send 10 items (909.0900%)margin
mn30cost USD: 1.70 to send 10 items (3090.9000%)margin
```
### Now these are off-chain micropayment scenarios
### performed Fri Jun 11 11:56:06 EDT 2021
```
Fri Jun 11 11:56:06 EDT 2021
TFuel(USD): 0.48622932399761
USD/Transaction: .15
0.01 TFuel in USD = .0049 x 1000 items = 4.9000 USD
visacost USD: 50.06 to send 1000 items (1021.6300%)margin
mn30cost USD: 150.00 to send 1000 items (3061.2200%)margin
mn30cost USD: .15 to send 1000 items (3.061200%)margin(1 service_payment)
```
```
Fri Jun 11 11:58:38 EDT 2021
TFuel(USD): 0.48603013940903
USD/Transaction: .15
0.001 TFuel in USD = .0005 x 10000 items = 5.0000 USD
visacost USD: 500.06 to send 10000 items (10001.2000%)margin
mn30cost USD: 1500.00 to send 10000 items (30000.0000%)margin
mn30cost USD: .15 to send 10000 items (3.000000%)margin(1 service_payment)
```
### Online newspaper use-case
#### Assumptions:
```
 WSJ.com example : $38.99 for 4 weeks = $1.3925 per day = $4.1775 per 3 days
 Subscriber reads 10 articles over 3 days on average.  Each article Header + blurb free
 Once article clicked on.  First min charged, additional min/read charged as scrolled to : avg 5 mins/article
 10 article * 5 min/read/article = 50 transactions over 3 days.
 If subsriber clicks on 0 articles over 3 days, no charges for that time
 Once user clicks on next article, new reserve fund is created for next 3 days
```

```
Fri Jun 11 12:22:15 EDT 2021
TFuel(USD): 0.48705293232693
USD/Transaction: .15
0.2 TFuel in USD = .0974 x 50 items = 4.8700 USD
visacost USD: 2.56 to send 50 items (52.5600%)margin
mn30cost USD: 7.50 to send 50 items (154.0000%)margin
mn30cost USD: .15 to send 50 items (3.080000%)margin(1 service_payment)
```
Daily Reader 3-4 articles/day
```
4.8700+0.15 x 10(3day periods/month) = $48.85/month 
```
Weekend Reader
```
4.8700+0.15 x 4(3day periods/month) = $19.54/month
```
Sporatic Reader : 2 articles/day on 10 days spread evenly across the month
```
0.963+0.15 x 10 = $9.78/month
```
Occasional Reader : 4 articles/month spread evenly across the month
```
0.4815+0.15 x 4 = $2.53/month
```
Rare Reader : 1 article/month
```
0.4815+0.15 x 1 = $0.63/month
```
Rare Reader Aborted Article : 1/5 article/month
```
0.0963+0.15 x 1 = $0.25/month
```