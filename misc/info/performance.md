TaskCollect performance statistics
==================================

TaskCollect is a piece of software for which performance is important, but may, for various reasons, not perform well. Unfortunately, the platforms that TaskCollect multiplexes have poor performance. TaskCollect, which retrieves data from these platforms, must unfortunately pass this issue onto the user. It cannot, unfortunately, magically fix the performance issue, which is on the source platforms' end.

This document collects statistics regarding various aspects of performance in TaskCollect.

Daymap
------

### Authentication

Daymap authentication for TaskCollect users of GIHS takes way too long. This is a result of EdPass, which is a single sign-on platform that handles authentication on Daymap's behalf. Unfortunately, EdPass has a shockingly under-performant implementation which:

  * Makes at least ten different HTTP redirects during the authentication flow
  * Requires sending at least seven HTTP requests before any credentials can be sent
  * Requires sending another four HTTP requests after credentials are actually sent
  * Returns several hundred kilobytes of data to the user per authentication attempt (in the browser, this figure is 10-30 times larger)
  * Is ridiculously overcomplicated for an authentication mechanism

The following table shows how slow Daymap authentication actually takes:

| Auth stage no. |   Trial 1 |   Trial 2 |   Trial 3 |   Trial 4 |   Trial 5 |   *Average* |
| :------------: | --------: | --------: | --------: | --------: | --------: | ----------: |
|    Stage 1     |    1124ms |     994ms |    1133ms |     945ms |     930ms |    *1025ms* |
|    Stage 2     |     112ms |     113ms |      67ms |      65ms |      99ms |      *91ms* |
|    Stage 3     |     203ms |     205ms |     528ms |     240ms |     297ms |     *295ms* |
|    Stage 4     |     272ms |     483ms |     427ms |     512ms |     259ms |     *391ms* |
|    Stage 5     |     264ms |     268ms |     308ms |     249ms |     570ms |     *332ms* |
|    Stage 6     |      62ms |      59ms |      61ms |      64ms |     109ms |      *71ms* |
|    Stage 7     |     617ms |     823ms |     652ms |     808ms |     629ms |     *706ms* |
|    Stage 8     |     558ms |     490ms |     516ms |     515ms |     592ms |     *534ms* |
|    Stage 9     |     980ms |     923ms |    1121ms |     974ms |    1021ms |    *1004ms* |
|    Stage 10    |     425ms |     409ms |     425ms |     560ms |     416ms |     *447ms* |
|    Stage 11    |     811ms |     921ms |     911ms |     820ms |     817ms |     *856ms* |
|    **TOTAL**   | **5.43s** | **5.69s** | **6.15s** | **5.76s** | **5.74s** | ***5.75s*** |

###### *(Results collected on 29 April 2023)*
