package telnet

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDial(t *testing.T) {
	server, err := NewMockServer()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		assert.NoError(t, server.Close())
		close(server.errors)
		for err := range server.errors {
			assert.NoError(t, err)
		}
	}()

	t.Run("connection refused", func(t *testing.T) {
		conn, err := Dial("127.0.0.2:12345", MockPassword)
		if !assert.Error(t, err) {
			// Close connection if established.
			assert.NoError(t, conn.Close())
		}

		assert.EqualError(t, err, "dial tcp 127.0.0.2:12345: connect: connection refused")
	})

	t.Run("authentication failed", func(t *testing.T) {
		conn, err := Dial(server.Addr(), "wrong")
		if !assert.Error(t, err) {
			assert.NoError(t, conn.Close())
		}

		assert.EqualError(t, err, "authentication failed")
	})

	t.Run("auth success", func(t *testing.T) {
		conn, err := Dial(server.Addr(), MockPassword, SetDialTimeout(5*time.Second))
		if assert.NoError(t, err) {
			assert.NoError(t, conn.Close())
		}
	})
}

func TestConn_Execute(t *testing.T) {
	server, err := NewMockServer()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		assert.NoError(t, server.Close())
		close(server.errors)
		for err := range server.errors {
			assert.NoError(t, err)
		}
	}()

	t.Run("incorrect command", func(t *testing.T) {
		conn, err := Dial(server.Addr(), MockPassword)
		if !assert.NoError(t, err) {
			return
		}
		defer assert.NoError(t, conn.Close())

		result, err := conn.Execute("")
		assert.Equal(t, err, ErrCommandEmpty)
		assert.Equal(t, 0, len(result))

		result, err = conn.Execute(string(make([]byte, 1001)))
		assert.Equal(t, err, ErrCommandTooLong)
		assert.Equal(t, 0, len(result))
	})

	t.Run("closed network connection", func(t *testing.T) {
		conn, err := Dial(server.Addr(), MockPassword)
		if !assert.NoError(t, err) {
			return
		}
		assert.NoError(t, conn.Close())

		result, err := conn.Execute(MockCommandHelp)
		assert.EqualError(t, err, fmt.Sprintf("write tcp %s->%s: use of closed network connection", conn.LocalAddr(), conn.RemoteAddr()))
		assert.Equal(t, 0, len(result))
	})

	t.Run("unknown command", func(t *testing.T) {
		conn, err := Dial(server.Addr(), MockPassword)
		if !assert.NoError(t, err) {
			return
		}
		defer func() {
			assert.NoError(t, conn.Close())
		}()

		result, err := conn.Execute("random")
		assert.NoError(t, err)
		assert.Equal(t, fmt.Sprintf("Please enter password:%s%s%s%s%s", CRLF, AuthSuccess, CRLF, "*** ERROR: unknown command 'random'", CRLF), result)
	})

	t.Run("success help command", func(t *testing.T) {
		conn, err := Dial(server.Addr(), MockPassword)
		if !assert.NoError(t, err) {
			return
		}
		defer func() {
			assert.NoError(t, conn.Close())
		}()

		result, err := conn.Execute(MockCommandHelp)
		assert.NoError(t, err)
		assert.Equal(t, fmt.Sprintf("Please enter password:%s%s%s%s%s", CRLF, AuthSuccess, CRLF, MockCommandHelpResponse, CRLF), result)
	})

	if run := getVar("TEST_7DTD_SERVER", "false"); run == "true" {
		addr := getVar("TEST_7DTD_SERVER_ADDR", "172.22.0.2:8081")
		password := getVar("TEST_7DTD_SERVER_PASSWORD", "banana")

		t.Run("7dtd server", func(t *testing.T) {
			needle := `*** List of Commands ***
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

			needle = strings.Replace(needle, "\n", "\r\n", -1)
			needle = strings.Replace(needle, "some generic info\r\n", "some generic info\n", -1)
			needle = strings.Replace(needle, "Also stores a list\r\n", "Also stores a list\n", -1)

			conn, err := Dial(addr, password)
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				assert.NoError(t, conn.Close())
			}()

			result, err := conn.Execute("help")
			assert.NoError(t, err)

			if !strings.Contains(result, needle) {
				diff := struct {
					R string
					N string
				}{R: result, N: needle}

				js, _ := json.Marshal(diff)
				fmt.Println(string(js))

				t.Error("response is not contain needle string")
			}
		})
	}
}

func TestConn_Interactive(t *testing.T) {
	server, err := NewMockServer()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		assert.NoError(t, server.Close())
		close(server.errors)
		for err := range server.errors {
			assert.NoError(t, err)
		}
	}()

	t.Run("unknown command", func(t *testing.T) {
		var r bytes.Buffer
		w := bytes.Buffer{}

		r.WriteString("random" + "\n")
		r.WriteString(ForcedExitCommand + "\n")

		err := DialInteractive(&r, &w, server.Addr(), MockPassword)
		if !assert.NoError(t, err) {
			return
		}

		assert.NoError(t, err)
		assert.Equal(t, fmt.Sprintf("Please enter password:%s%s%s%s%s", CRLF, AuthSuccess, CRLF, "*** ERROR: unknown command 'random'", CRLF), w.String())
	})

	t.Run("success help command", func(t *testing.T) {
		var r bytes.Buffer
		w := bytes.Buffer{}

		r.WriteString(MockCommandHelp + "\n")
		r.WriteString(ForcedExitCommand + "\n")

		err := DialInteractive(&r, &w, server.Addr(), MockPassword, SetExitCommand("exit"))
		if !assert.NoError(t, err) {
			return
		}

		assert.NoError(t, err)
		assert.Equal(t, fmt.Sprintf("Please enter password:%s%s%s%s%s", CRLF, AuthSuccess, CRLF, MockCommandHelpResponse, CRLF), w.String())
	})
}

// getVar returns environment variable or default value.
func getVar(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}
