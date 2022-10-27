package cmd

import (
	"errors"
	"fmt"
	"math"
	"math/rand"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"github.com/yugabyte/gocql"
)

const (
	// ScaleFactor is the default scale factor
	ScaleFactor = 6
	// ParalleTasks is the default parallel tasks
	ParallelTasks = 16
	// Timeout is the default timeout, in ms. Higher than gocql default, because scan queries generally take longer
	Timeout = 1500

	// ErrCountMax is the default number of errors we report on a single table scan
	ErrCountMax = 5
)

var (
	Version = "DEV"

	debug    bool
	errored  bool
	errCount int

	cluster       *gocql.ClusterConfig
	hosts         []string
	tables        []string
	tasksPerTable int

	// Scaling factor of tasks per table
	scaleFactor int

	// parallel connections/queries
	parallelTasks int

	// the time in miliseconds a query must succeed in
	timeout int

	// max printed errors
	errmax int

	certPath, keyPath, caPath string
	enableHostVerification    bool

	user, password string

	rootCmd = &cobra.Command{
		Use:     "ycrc <keyspace>",
		Args:    cobra.ExactArgs(1),
		Short:   "YCql Row Count",
		Long:    "YCql Row Count (ycrc) parallelizes counting the number of rows in a table for YugabyteDB CQL, allowing count(*) on tables that otherwise would fail with query timeouts",
		Version: Version,
		Run: func(cmd *cobra.Command, args []string) {
			rowCount(args[0])
		},
	}
)

// partitionMap is the table and range of a partition to be scanned
type partitionMap struct {
	table    string
	pColumns string
	lbound   int
	ubound   int
}

func init() {
	rootCmd.Flags().BoolVarP(&debug, "debug", "d", false, "Verbose logging")

	rootCmd.Flags().StringSliceVarP(&hosts, "hosts", "c", []string{"127.0.0.1"}, "Cluster to connect to")
	rootCmd.Flags().StringSliceVar(&tables, "tables", []string{}, "List of tables inside of the keyspace - default to all")
	rootCmd.Flags().IntVarP(&scaleFactor, "scale", "s", ScaleFactor, "Scaling factor of tasks per table, an int between 1 and 10")
	rootCmd.Flags().IntVarP(&parallelTasks, "parallel", "p", ParallelTasks, "Number of concurrent tasks")
	rootCmd.Flags().IntVarP(&timeout, "timeout", "t", Timeout, "Timeout of a single query, in ms")
	rootCmd.Flags().IntVar(&errmax, "maxerrprinted", ErrCountMax, "Max errors to print - 0 to unset")

	// SSL certs
	rootCmd.Flags().StringVar(&certPath, "sslcert", "", "SSL cert path")
	rootCmd.Flags().StringVar(&keyPath, "sslkey", "", "SSL key path")
	rootCmd.Flags().StringVar(&caPath, "sslca", "", "SSL root ca path")
	rootCmd.Flags().BoolVar(&enableHostVerification, "verify", false, "Strictly verify SSL host (off by default)")

	// auth
	rootCmd.Flags().StringVarP(&user, "user", "u", "cassandra", "database user")
	rootCmd.Flags().StringVar(&password, "password", "", "user password")

}

// Execute runs the program
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func parallelRowCount(pMaps chan partitionMap, result chan int, done chan struct{}, session *gocql.Session) {
	for pMap := range pMaps {
		// TODO: improve error handling
		count, _ := checkPartitionRowCount(pMap, session)
		select {
		case result <- count:
		case <-done:
			return
		}

	}

}

func checkPartitionRowCount(pMap partitionMap, session *gocql.Session) (int, error) {
	statement := fmt.Sprintf(`SELECT count(*) as rows from %s.%s WHERE partition_hash(%s) >= ? AND partition_hash(%s) <= ?`, cluster.Keyspace, pMap.table, pMap.pColumns, pMap.pColumns)
	var rows int
	if debug {
		fmt.Printf("DEBUG: executing `SELECT count(*) as rows from %s.%s WHERE partition_hash(%s) >= %d and partition_hash(%s) <= %d`\n", cluster.Keyspace, pMap.table, pMap.pColumns, pMap.lbound, pMap.pColumns, pMap.ubound)
	}
	err := session.Query(statement, pMap.lbound, pMap.ubound).Scan(&rows)

	if err != nil {
		// always print error to debug log
		if debug {
			fmt.Printf("DEBUG: Unable to get row count from: %s.%s partition hash: %s between %d and %d\n", cluster.Keyspace, pMap.table, pMap.pColumns, pMap.lbound, pMap.ubound)
			fmt.Println(err)
		}

		if errmax == 0 || errCount < errmax {
			fmt.Fprintf(os.Stderr, "Unable to get row count from: %s.%s partition hash: %s between %d and %d\n", cluster.Keyspace, pMap.table, pMap.pColumns, pMap.lbound, pMap.ubound)
			fmt.Fprintln(os.Stderr, err)
			fmt.Fprintf(os.Stderr, "--------\nWARNING: Program will continue but rowcount is inaccurate for this table\n--------\n")
			errCount = errCount + 1
			errored = true
		}
	}

	return rows, err
}

func checkTableRowCounts(table string, session *gocql.Session) error {
	// new table, new err count
	errCount = 0

	fmt.Printf("Checking row counts for: %s.%s\n", cluster.Keyspace, table)

	rows := session.Query(`SELECT column_name, position FROM system_schema.columns WHERE keyspace_name = ? AND table_name = ? AND kind='partition_key'`, cluster.Keyspace, table).Iter()

	/*
	   TODO: There may be a max number of partition_keys for a table - perhaps
	   256, though this is just a legacy number from the original python script.

	   If so, it's possible to simply create a slice of correct size, and then
	   order into the slice from the rows directly, rather than create a map
	   first. The only logic behind the map was ensuring that we had a growable
	   space for all items returned from the query, even for massive table sizes.

	   If YCQL provides limits on partition_keys in a table, we can just create
	   a slice of size MAX_PARTITION_KEYS and be done with it.
	*/
	partitionColumnsMap := make(map[int]string)

	var columnName string
	var position int

	for rows.Scan(&columnName, &position) {
		partitionColumnsMap[position] = columnName

	}
	if err := rows.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "Error on getting partition columns in of table: %s.%s\n", cluster.Keyspace, table)
		fmt.Fprintln(os.Stderr, err)
		errored = true
	}
	partitionColumns := make([]string, len(partitionColumnsMap))
	for p, c := range partitionColumnsMap {
		partitionColumns[p] = c
	}

	pColumns := strings.Join(partitionColumns, ",")
	fmt.Printf("Partitioning columns for %s.%s:(%s)\n", cluster.Keyspace, table, pColumns)

	fmt.Printf("Performing %d checks for %s.%s with %d parallel tasks\n", tasksPerTable, cluster.Keyspace, table, parallelTasks)

	// This is sort of ugly, but it's useful to have rangeSize for calculating ubound
	// later, and shuffling lbounds is also required just below.
	rangeSize := 64 * 1024 / tasksPerTable
	var lbounds []int

	for i := 0; i < tasksPerTable; i++ {
		lbounds = append(lbounds, i*rangeSize)
	}

	// we want to avoid just iterating through single tablet servers, so randomize the
	// lbounds array.
	lbounds = shuffle(lbounds)

	var wg sync.WaitGroup
	done := make(chan struct{})
	defer close(done)

	pMaps := make(chan partitionMap)
	results := make(chan int)

	// get timing
	start := time.Now()

	wg.Add(parallelTasks)

	for i := 0; i < parallelTasks; i++ {
		go func() {
			parallelRowCount(pMaps, results, done, session)
			wg.Done()
		}()
	}
	go func() {
		// all pMaps have been sent, close pMaps to trigger end of goroutine execution
		defer close(pMaps)
		for _, lbound := range lbounds {
			ubound := lbound + rangeSize - 1
			pMaps <- partitionMap{table, pColumns, lbound, ubound}
		}
	}()

	// close the results channel once we're done
	go func() {
		wg.Wait()
		close(results)
	}()

	var count int
	for r := range results {
		count += r
	}

	elapsed := time.Since(start).Milliseconds()

	fmt.Printf("==========\nTotal time: %d ms\n==========\n", elapsed)

	fmt.Printf("Total Row Count %s.%s = %d\n\n", cluster.Keyspace, table, count)
	if errored {
		return errors.New("Error during parallel execution: row counts may be inaccurate")
	}
	return nil
}

func checkKeyspaceTableRowCounts(session *gocql.Session) error {
	fmt.Printf("Checking table row counts for keyspace: %s\n", cluster.Keyspace)

	//	var tables []string
	if len(tables) == 0 {
		rows := session.Query(`SELECT table_name FROM system_schema.tables where keyspace_name = ?`, cluster.Keyspace).Iter()
		var table string
		for rows.Scan(&table) {
			tables = append(tables, table)
		}
		if err := rows.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Error on getting tables in keyspace: %s, cannot continue\n", cluster.Keyspace)
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
	} else {
		var verifiedTables []string
		for _, table := range tables {
			rows := session.Query(`SELECT table_name FROM system_schema.tables where keyspace_name = ? and table_name = ?`, cluster.Keyspace, table).Iter()
			if rows.NumRows() == 0 {
				fmt.Fprintf(os.Stderr, "Could not find table %s in keyspace: %s, skipping\n", table, cluster.Keyspace)
				continue
			}
			if err := rows.Close(); err != nil {
				fmt.Fprintf(os.Stderr, "Error on validating tables in keyspace: %s, cannot continue\n", cluster.Keyspace)
				fmt.Fprintln(os.Stderr, err)
				os.Exit(2)
			}
			verifiedTables = append(verifiedTables, table)
		}
		tables = verifiedTables
	}

	if len(tables) == 0 {
		fmt.Fprintf(os.Stdout, "No tables selected in keyspace %s, exiting\n", cluster.Keyspace)
		os.Exit(0)
	}

	for _, table := range tables {
		err := checkTableRowCounts(table, session)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error getting row counts for table: %s.%s\n", cluster.Keyspace, table)
			return err
		}
	}

	return nil
}

func rowCount(keyspace string) {

	// ssl cert requires key as well
	if !(certPath == "" && keyPath == "") {
		if certPath == "" || keyPath == "" {
			fmt.Fprintf(os.Stderr, "If sslcert or sslkey is specified, both arguments are required\n")
			os.Exit(1)
		}
	}

	if !(1 <= scaleFactor && scaleFactor <= 10) {
		fmt.Fprintf(os.Stderr, "Scaling factor must be between 1 and 10, setting to the default\n")
		scaleFactor = ScaleFactor
	}

	if parallelTasks < 1 {
		fmt.Fprintf(os.Stderr, "There must be at least 1 parallel process, setting to default\n")
		parallelTasks = ParallelTasks

	}

	// arbitrary
	if timeout < 100 {
		fmt.Fprintf(os.Stderr, "Timeout must be greater than 100ms, setting to default\n")
		timeout = Timeout
	}
	if errmax < 0 {
		fmt.Fprintf(os.Stderr, "maxerrprinted must be 0 (no trim) or positive, setting to default\n")
		errmax = ErrCountMax
	}

	// 128, 256, 512, 1025, 2048, 4096, 8192, 16384, 32768, 65536
	tasksPerTable = int(4096 * math.Exp2(float64(scaleFactor)-6))

	cluster = gocql.NewCluster(hosts...)

	if password != "" {
		cluster.Authenticator = gocql.PasswordAuthenticator{
			Username: user,
			Password: password,
		}
	}

	cluster.Timeout = time.Duration(timeout) * time.Millisecond

	if certPath != "" || keyPath != "" || caPath != "" {
		cluster.SslOpts = &gocql.SslOptions{
			CertPath:               certPath,
			KeyPath:                keyPath,
			CaPath:                 caPath,
			EnableHostVerification: enableHostVerification,
		}
	}
	cluster.Keyspace = keyspace
	session, err := cluster.CreateSession()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating cluster session, cannot continue\n%v\n", err)
		os.Exit(2)
	}
	defer session.Close()

	err = checkKeyspaceTableRowCounts(session)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting row count on some tables, failing")
		fmt.Fprintln(os.Stderr, err)
	}

}

func shuffle(vals []int) []int {
	r := rand.New(rand.NewSource(time.Now().Unix()))

	ret := make([]int, len(vals))
	perm := r.Perm(len(vals))

	for i, randIdx := range perm {
		ret[i] = vals[randIdx]
	}
	return ret
}
