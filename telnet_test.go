package telnet_test

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gorcon/telnet"
	"github.com/gorcon/telnet/telnettest"
)

func authHandler(c *telnettest.Context) {
	switch c.Request() {
	case c.Server().Settings.Password:
		c.Writer().WriteString(telnet.ResponseAuthSuccess + telnet.CRLF + telnet.CRLF + telnet.CRLF + telnet.CRLF)
		c.Writer().WriteString(telnettest.AuthSuccessWelcomeMessage + telnet.CRLF + telnet.CRLF)

		c.Auth.Success = true
		c.Auth.Break = true
	case "unexpect":
		c.Writer().WriteString("My spoon is too big" + telnet.CRLF + telnet.CRLF)

		c.Auth.Success = false
		c.Auth.Break = true
	default:
		c.Writer().WriteString(telnet.ResponseAuthIncorrectPassword + telnet.CRLF)
	}
}

func commandHandler(c *telnettest.Context) {
	switch c.Request() {
	case "", "exit":
	case "help":
		c.Writer().WriteString(fmt.Sprintf("2020-11-14T23:09:20 31220.643 "+telnet.ResponseINFLayout, c.Request(), c.Conn().RemoteAddr()) + telnet.CRLF)
		c.Writer().WriteString("lorem ipsum dolor sit amet" + telnet.CRLF)
	default:
		c.Writer().WriteString(fmt.Sprintf("*** ERROR: unknown command '%s'", c.Request()) + telnet.CRLF)
	}

	c.Writer().Flush()
}

func TestDial(t *testing.T) {
	server := telnettest.NewServer(
		telnettest.SetSettings(telnettest.Settings{Password: "password"}),
		telnettest.SetAuthHandler(authHandler),
	)
	defer server.Close()

	t.Run("connection refused", func(t *testing.T) {
		wantErrContains := "connect: connection refused"

		_, err := telnet.Dial("127.0.0.2:12345", "password")
		if err == nil || !strings.Contains(err.Error(), wantErrContains) {
			t.Errorf("got err %q, want to contain %q", err, wantErrContains)
		}
	})

	t.Run("incorrect password", func(t *testing.T) {
		_, err := telnet.Dial(server.Addr(), string(make([]byte, 1001)))
		if !errors.Is(err, telnet.ErrCommandTooLong) {
			t.Errorf("got err %q, want %q", err, telnet.ErrCommandTooLong)
		}
	})

	t.Run("authentication failed", func(t *testing.T) {
		_, err := telnet.Dial(server.Addr(), "wrong")
		if !errors.Is(err, telnet.ErrAuthFailed) {
			t.Errorf("got err %q, want %q", err, telnet.ErrAuthFailed)
		}
	})

	t.Run("unexpected auth response", func(t *testing.T) {
		_, err := telnet.Dial(server.Addr(), "unexpect")
		if !errors.Is(err, telnet.ErrAuthUnexpectedMessage) {
			t.Errorf("got err %q, want %q", err, telnet.ErrAuthUnexpectedMessage)
		}
	})

	t.Run("auth success", func(t *testing.T) {
		conn, err := telnet.Dial(server.Addr(), "password", telnet.SetDialTimeout(5*time.Second))
		if err != nil {
			t.Errorf("got err %q, want %v", err, nil)
			return
		}
		defer conn.Close()

		if conn.Status() != telnettest.AuthSuccessWelcomeMessage {
			t.Fatalf("got result %q, want %q", conn.Status(), telnettest.AuthSuccessWelcomeMessage)
		}
	})
}

func TestConn_Execute(t *testing.T) {
	server := telnettest.NewServer(
		telnettest.SetSettings(telnettest.Settings{Password: "password"}),
		telnettest.SetAuthHandler(authHandler),
		telnettest.SetCommandHandler(commandHandler),
	)
	defer server.Close()

	t.Run("incorrect command", func(t *testing.T) {
		conn, err := telnet.Dial(server.Addr(), "password")
		if err != nil {
			t.Fatalf("got err %q, want %v", err, nil)
		}
		defer conn.Close()

		result, err := conn.Execute("")
		if !errors.Is(err, telnet.ErrCommandEmpty) {
			t.Errorf("got err %q, want %q", err, telnet.ErrCommandEmpty)
		}

		if len(result) != 0 {
			t.Fatalf("got result len %d, want %d", len(result), 0)
		}

		result, err = conn.Execute(string(make([]byte, 1001)))
		if !errors.Is(err, telnet.ErrCommandTooLong) {
			t.Errorf("got err %q, want %q", err, telnet.ErrCommandTooLong)
		}

		if len(result) != 0 {
			t.Fatalf("got result len %d, want %d", len(result), 0)
		}
	})

	t.Run("closed network connection", func(t *testing.T) {
		conn, err := telnet.Dial(server.Addr(), "password")
		if err != nil {
			t.Fatalf("got err %q, want %v", err, nil)
		}
		conn.Close()

		result, err := conn.Execute("help")
		wantErrMsg := fmt.Sprintf("write tcp %s->%s: use of closed network connection", conn.LocalAddr(), conn.RemoteAddr())
		if err == nil || err.Error() != wantErrMsg {
			t.Errorf("got err %q, want to contain %q", err, wantErrMsg)
		}

		if len(result) != 0 {
			t.Fatalf("got result len %d, want %d", len(result), 0)
		}
	})

	t.Run("unknown command", func(t *testing.T) {
		conn, err := telnet.Dial(server.Addr(), "password")
		if err != nil {
			t.Fatalf("got err %q, want %v", err, nil)
		}
		defer conn.Close()

		result, err := conn.Execute("random")
		if err != nil {
			t.Fatalf("got err %q, want %v", err, nil)
		}

		resultWant := "*** ERROR: unknown command 'random'"
		if result != resultWant {
			t.Fatalf("got result %q, want %q", result, resultWant)
		}
	})

	t.Run("success help command", func(t *testing.T) {
		conn, err := telnet.Dial(server.Addr(), "password", telnet.SetClearResponse(true))
		if err != nil {
			t.Fatalf("got err %q, want %v", err, nil)
		}
		defer conn.Close()

		result, err := conn.Execute("help")
		if err != nil {
			t.Fatalf("got err %q, want %v", err, nil)
		}

		resultWant := "lorem ipsum dolor sit amet"
		if result != resultWant {
			t.Fatalf("got result %q, want %q", result, resultWant)
		}
	})

	t.Run("multiple commands", func(t *testing.T) {
		conn, err := telnet.Dial(server.Addr(), "password", telnet.SetClearResponse(true))
		if err != nil {
			t.Fatalf("got err %q, want %v", err, nil)
		}
		defer conn.Close()

		// Command 1
		result, err := conn.Execute("help")
		if err != nil {
			t.Fatalf("got err %q, want %v", err, nil)
		}

		resultWant := "lorem ipsum dolor sit amet"
		if result != resultWant {
			t.Fatalf("got result %q, want %q", result, resultWant)
		}

		// Command 2
		result, err = conn.Execute("random")
		if err != nil {
			t.Fatalf("got err %q, want %v", err, nil)
		}

		resultWant = "*** ERROR: unknown command 'random'"
		if result != resultWant {
			t.Fatalf("got result %q, want %q", result, resultWant)
		}

		// Command 3
		result, err = conn.Execute("help")
		if err != nil {
			t.Fatalf("got err %q, want %v", err, nil)
		}

		resultWant = "lorem ipsum dolor sit amet"
		if result != resultWant {
			t.Fatalf("got result %q, want %q", result, resultWant)
		}
	})

	if run := getVar("TEST_7DTD_SERVER", "false"); run == "true" {
		addr := getVar("TEST_7DTD_SERVER_ADDR", "172.22.0.2:8081")
		password := getVar("TEST_7DTD_SERVER_PASSWORD", "banana")

		t.Run("7dtd server", func(t *testing.T) {
			conn, err := telnet.Dial(addr, password, telnet.SetClearResponse(true))
			if err != nil {
				t.Fatal(err)
			}
			defer conn.Close()

			// Command 1
			needle := func() string {
				n := `*** Generic Console Help ***
To get further help on a specific topic or command type (without the brackets)
    help <topic / command>

Generic notation of command parameters:
   <param name>              Required parameter
   <entityId / player name>  Possible types of parameter values
   [param name]              Optional parameter

*** List of Help Topics ***
None yet

*** List of Commands ***
 admin => Manage user permission levels
 aiddebug => Toggles AIDirector debug output.
 audio => Watch audio stats
 automove => Player auto movement
 ban => Manage ban entries
 bents => Switches block entities on/off
 BiomeParticles => Debug
 buff => Applies a buff to the local player
 buffplayer => Apply a buff to a player
 chunkcache cc => shows all loaded chunks in cache
 chunkobserver co => Place a chunk observer on a given position.
 chunkreset cr => resets the specified chunks
 commandpermission cp => Manage command permission levels
 creativemenu cm => enables/disables the creativemenu
 debuff => Removes a buff from the local player
 debuffplayer => Remove a buff from a player
 debugmenu dm => enables/disables the debugmenu ` + `
 debugshot dbs => Lets you make a screenshot that will have some generic info
on it and a custom text you can enter. Also stores a list
of your current perk levels in a CSV file next to it.
 debugweather => Dumps internal weather state to the console.
 decomgr => ` + `
 dms => Gives control over Dynamic Music functionality.
 dof => Control DOF
 enablescope es => toggle debug scope
 exhausted => Makes the player exhausted.
 exportcurrentconfigs => Exports the current game config XMLs
 exportprefab => Exports a prefab from a world area
 floatingorigin fo => ` + `
 fov => Camera field of view
 gamestage => usage: gamestage - displays the gamestage of the local player.
 getgamepref gg => Gets game preferences
 getgamestat ggs => Gets game stats
 getoptions => Gets game options
 gettime gt => Get the current game time
 gfx => Graphics commands
 givequest => usage: givequest questname
 giveself => usage: giveself itemName [qualityLevel=6] [count=1] [putInInventory=false] [spawnWithMods=true]
 giveselfxp => usage: giveselfxp 10000
 help => Help on console and specific commands
 kick => Kicks user with optional reason. "kick playername reason"
 kickall => Kicks all users with optional reason. "kickall reason"
 kill => Kill a given entity
 killall => Kill all entities
 lgo listgameobjects => List all active game objects
 lights => Debug views to optimize lights
 listents le => lists all entities
 listplayerids lpi => Lists all players with their IDs for ingame commands
 listplayers lp => lists all players
 listthreads lt => lists all threads
 loggamestate lgs => Log the current state of the game
 loglevel => Telnet/Web only: Select which types of log messages are shown
 mem => Prints memory information and unloads resources or changes garbage collector
 memcl => Prints memory information on client and calls garbage collector
 occlusion => Control OcclusionManager
 pirs => tbd
 pois => Switches distant POIs on/off
 pplist => Lists all PersistentPlayer data
 prefab => ` + `
 prefabupdater => ` + `
 profilenetwork => Writes network profiling information
 profiling => Enable Unity profiling for 300 frames
 removequest => usage: removequest questname
 repairchunkdensity rcd => check and optionally fix densities of a chunk
 saveworld sa => Saves the world manually.
 say => Sends a message to all connected clients
 ScreenEffect => Sets a screen effect
 setgamepref sg => sets a game pref
 setgamestat sgs => sets a game stat
 settargetfps => Set the target FPS the game should run at (upper limit)
 settempunit stu => Set the current temperature units.
 settime st => Set the current game time
 show => Shows custom layers of rendering.
 showalbedo albedo => enables/disables display of albedo in gBuffer
 showchunkdata sc => shows some date of the current chunk
 showClouds => Artist command to show one layer of clouds.
 showhits => Show hit entity locations
 shownexthordetime => Displays the wandering horde time
 shownormals norms => enables/disables display of normal maps in gBuffer
 showspecular spec => enables/disables display of specular values in gBuffer
 showswings => Show melee swing arc rays
 shutdown => shuts down the game
 sleeper => Show sleeper info
 smoothworldall swa => Applies some batched smoothing commands.
 sounddebug => Toggles SoundManager debug output.
 spawnairdrop => Spawns an air drop
 spawnentity se => spawns an entity
 spawnentityat sea => Spawns an entity at a give position
 spawnscouts => Spawns zombie scouts
 SpawnScreen => Display SpawnScreen
 spawnsupplycrate => Spawns a supply crate where the player is
 spawnwanderinghorde spawnwh => Spawns a wandering horde of zombies
 spectator spectatormode sm => enables/disables spectator mode
 spectrum => Force a particular lighting spectrum.
 stab => stability
 starve hungry food => Makes the player starve (optionally specify the amount of food you want to have in percent).
 switchview sv => Switch between fpv and tpv
 SystemInfo => List SystemInfo
 teleport tp => Teleport the local player
 teleportplayer tele => Teleport a given player
 thirsty water => Makes the player thirsty (optionally specify the amount of water you want to have in percent).
 traderarea => ...
 trees => Switches trees on/off
 updatelighton => Commands for UpdateLightOnAllMaterials and UpdateLightOnPlayers
 version => Get the currently running version of the game and loaded mods
 visitmap => Visit an given area of the map. Optionally run the density check on each visited chunk.
 water => Control water settings
 weather => Control weather settings
 weathersurvival => Enables/disables weather survival
 whitelist => Manage whitelist entries
 wsmats workstationmaterials => Set material counts on workstations.
 xui => Execute XUi operations
 xuireload => Access xui related functions such as reinitializing a window group, opening a window group
 zip => Control zipline settings`

				n = strings.Replace(n, "\n", "\r\n", -1)
				n = strings.Replace(n, "some generic info\r\n", "some generic info\n", -1)
				n = strings.Replace(n, "Also stores a list\r\n", "Also stores a list\n", -1)

				return n
			}()
			result, err := conn.Execute("help")
			if err != nil {
				t.Fatalf("got err %q, want %v", err, nil)
			}

			if result != needle {
				t.Fatalf("got result %q, want %q", result, needle)
			}

			// Command 2
			needle = "*** ERROR: unknown command 'status'"
			result, err = conn.Execute("status")
			if err != nil {
				t.Fatalf("got err %q, want %v", err, nil)
			}

			if result != needle {
				t.Fatalf("got result %q, want %q", result, needle)
			}

			// Command 3
			needle = "INF Chat (from '-non-player-', entity id '-1', to 'Global'): 'Server': 10"
			result, err = conn.Execute("say 10")
			if err != nil {
				t.Fatalf("got err %q, want %v", err, nil)
			}

			if !strings.Contains(needle, needle) {
				t.Fatalf("got result %q, want to contain %q", result, needle)
			}
		})
	}
}

func TestConn_Interactive(t *testing.T) {
	server := telnettest.NewUnstartedServer()
	server.Settings.Password = "password"
	server.SetAuthHandler(authHandler)
	server.SetCommandHandler(commandHandler)
	server.Start()
	defer server.Close()

	t.Run("connection refused", func(t *testing.T) {
		wantErrContains := "connect: connection refused"

		r, w := bytes.Buffer{}, bytes.Buffer{}

		err := telnet.DialInteractive(&r, &w, "127.0.0.2:12345", "password")
		if err == nil || !strings.Contains(err.Error(), wantErrContains) {
			t.Errorf("got err %q, want to contain %q", err, wantErrContains)
		}
	})

	t.Run("unknown command", func(t *testing.T) {
		needle := telnet.ResponseEnterPassword + telnet.CRLF +
			telnet.ResponseAuthSuccess + telnet.CRLF + telnet.CRLF + telnet.CRLF + telnet.CRLF +
			telnettest.AuthSuccessWelcomeMessage + telnet.CRLF + telnet.CRLF +
			"*** ERROR: unknown command 'random'" + telnet.CRLF

		r, w := bytes.Buffer{}, bytes.Buffer{}

		r.WriteString("random" + "\n")
		r.WriteString(telnet.ForcedExitCommand + "\n")

		err := telnet.DialInteractive(&r, &w, server.Addr(), "password")
		if err != nil {
			t.Fatalf("got err %q, want %v", err, nil)
		}

		if w.String() != needle {
			t.Fatalf("got result %q, want %q", w.String(), needle)
		}
	})

	t.Run("success help command", func(t *testing.T) {
		// TODO: server.Addr() in needle must be client address.
		// This is impossible to check in current TELNET implementation.
		needle := telnet.ResponseEnterPassword + telnet.CRLF +
			telnet.ResponseAuthSuccess + telnet.CRLF + telnet.CRLF + telnet.CRLF + telnet.CRLF +
			telnettest.AuthSuccessWelcomeMessage + telnet.CRLF + telnet.CRLF +
			fmt.Sprintf("2020-11-14T23:09:20 31220.643 "+telnet.ResponseINFLayout, "help", server.Addr()) + telnet.CRLF +
			"lorem ipsum dolor sit amet" + telnet.CRLF

		r, w := bytes.Buffer{}, bytes.Buffer{}

		r.WriteString("help" + "\n")
		r.WriteString(telnet.ForcedExitCommand + "\n")

		err := telnet.DialInteractive(&r, &w, server.Addr(), "password", telnet.SetExitCommand("exit"))
		if err != nil {
			t.Fatalf("got err %q, want %v", err, nil)
		}

		if !strings.Contains(w.String(), "help") {
			t.Fatalf("got result %q, want to contain %q", w.String(), needle)
		}
	})
}

// getVar returns environment variable or default value.
func getVar(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}
