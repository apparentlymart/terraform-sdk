provider "test" {
}

resource "test_instance" "test" {
  type  = "z4.weedy"
  image = "img-abc123"

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
