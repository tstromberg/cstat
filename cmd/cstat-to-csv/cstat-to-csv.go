// cstat-to-csv collects multiple cstat results into a single CSV file
package main

import (
	"flag"
)

var heading = flag.String("heading", true, "show header")

/*    <heading>
<mtime>
<average>
--
<heading>
<results>
*/

func main() {
	flag.Parse()
}
