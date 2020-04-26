package main

import (
	"log"
	"encoding/json"
	"github.com/boltdb/bolt"
)

/*
Database for ArOZ Online make use of the keydb by robaho
See more details of this database in https://www.opsdash.com/blog/persistent-key-value-store-golang.html

Why the developer choose this DB you might ask? 
Beacuse it is simple and simple is beautiful :)
*/

//Initiate the database object
func system_db_service_init(dbfile string) *bolt.DB{
	db, err := bolt.Open(dbfile, 0600, nil)
	if err != nil {
		log.Fatal(err)
	} 

	//Create the central system db for all system services
	err = db.Update(func(tx *bolt.Tx) error {
		_, err = tx.CreateBucketIfNotExists([]byte("SYSTEM"))
        if err != nil {
        	return err
		}
		
        return nil
	})
	
	if (err != nil){
		log.Fatal(err)
	}

	log.Println("Open Ticket Support Key-value Database Service Loaded");
	return db;
}

/*
	Create / Drop a table
	Usage:
	err := system_db_newTable(sysdb, "MyTable")
	err := system_db_dropTable(sysdb, "MyTable")
*/
func system_db_newTable(dbObject *bolt.DB, tableName string) error{
	err := dbObject.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(tableName))
		if err != nil {
			return err
		}
		return nil
	})
	return err
}

func system_db_dropTable(dbObject *bolt.DB, tableName string) error{
	err := dbObject.Update(func(tx *bolt.Tx) error {
		err := tx.DeleteBucket([]byte(tableName))
		if err != nil {
			return err
		}
		return nil
	})
	return err
}
/*
	Write to database with given tablename and key. Example Usage:
	type demo struct{
		content string
	}
	thisDemo := demo{
		content: "Hello World",
	}
	err := system_db_write(sysdb, "MyTable", "username/message",thisDemo);
*/
func system_db_write(dbObject *bolt.DB, tableName string, key string, value interface{}) error{
	jsonString, err := json.Marshal(value);
	if (err != nil){
		return err
	}
	err = dbObject.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(tableName))
		b := tx.Bucket([]byte(tableName))
		err = b.Put([]byte(key), jsonString)
		return err
	})
	return err
}

/*
	Read from database and assign the content to a given datatype. Example Usage:

	type demo struct{
		content string
	}
	thisDemo := new(demo)
	err := system_db_write(sysdb, "MyTable", "username/message",&thisDemo);
*/
func system_db_read(dbObject *bolt.DB, tableName string, key string, assignee interface{}) error{
	err := dbObject.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(tableName))
		v := b.Get([]byte(key))
		json.Unmarshal(v, &assignee)
		return nil
	})
	return err
}


func system_db_listTable(dbObject *bolt.DB, table string) [][][]byte{
	var results [][][]byte
	dbObject.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(table))
		c := b.Cursor()
		
		for k, v := c.First(); k != nil; k, v = c.Next() {
			results = append(results, [][]byte{k, v})
		}
		return nil
	})
	return results;
}


func system_db_closeDatabase(dbObject *bolt.DB){
	dbObject.Close()
	return;
}

