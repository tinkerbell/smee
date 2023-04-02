/*
standalone contains a standalone backend for boots so it can run without
a packet or tinkerbell backend. Instead of using a scalable backend, hardware data
is loaded from a json file and stored in a list in memory.

TODO:
   * only supports one interface right now, multiple can be defined but might act weird
   * methods only return the first interface's information and ignore everything else
   * methods don't have godoc yet and since this is for test maybe never will?
*/

package standalone
