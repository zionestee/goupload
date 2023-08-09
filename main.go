package goupload

import (
	"os"

	"github.com/globalsign/mgo"
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
)

func main() {
	app := fiber.New()

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Hello, World!")
	})

	app.Get("/file/:id", func(c *fiber.Ctx) error {
		fileId := c.Params("id")

		fileBytes, err := os.ReadFile("./data/" + fileId)
		if err != nil {
			panic(err)
		}

		c.Status(fiber.StatusOK)
		c.Set("Content-Type", "application/octet-stream")
		c.Write(fileBytes)
		return c.Status(fiber.StatusCreated).JSON(&fiber.Map{
			"data": "Upload complete",
		})
	})
	app.Delete("/file/:id", func(c *fiber.Ctx) error {
		const (
			mogoDBEnPint = "mongodb://localhost:27017"
			DBName       = "luzio-upload"
			DBCollection = "files"
		)
		ConnectionDB, err := mgo.Dial(mogoDBEnPint)
		if err != nil {
			return c.Status(fiber.StatusCreated).JSON(&fiber.Map{
				"error": err.Error(),
			})
		}

		fileId := c.Params("id")

		collection := ConnectionDB.DB(DBName).C(DBCollection)

		err = collection.Remove(bson.M{"key": fileId})
		if err != nil {
			return c.Status(fiber.StatusCreated).JSON(&fiber.Map{
				"error": err.Error(),
			})
		}

		os.Remove("./data/" + fileId)
		os.Remove("./data/" + fileId + ".info")

		return c.Status(fiber.StatusCreated).JSON(&fiber.Map{
			"data": "Deleted",
		})
	})

	app.Listen(":8888")
}
