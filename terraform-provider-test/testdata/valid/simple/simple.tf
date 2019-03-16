provider "test" {
}

data "test_echo" "test" {
  given = "img-abc123"

  dynamic = {}
}

resource "test_instance" "test" {
  type  = "z4.weedy"
  image = data.test_echo.test.result

  access {
    policy = {
      statements: [
        {
          action: "foo:Create",
          principal: {
            service: "foo.example.com",
          },
        },
      ]
    }
  }

  network_interface "foo" {
  }
  network_interface "bar" {
    create_public_addrs = false
  }
}
