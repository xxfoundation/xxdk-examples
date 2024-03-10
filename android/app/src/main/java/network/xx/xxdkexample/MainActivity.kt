package network.xx.xxdkexample

import androidx.appcompat.app.AppCompatActivity
import android.os.Bundle
import android.util.Log
import android.widget.TextView
import androidx.lifecycle.Observer
import bindings.Bindings.newCmix;
import com.lyft.kronos.AndroidClockFactory
import com.lyft.kronos.KronosClock
import java.io.File
import java.nio.file.Path
import java.util.Base64

class MainActivity : AppCompatActivity() {

    lateinit var net: bindings.Cmix
    lateinit var DM: bindings.DMClient

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_main)


        val context = this.applicationContext;

        // xxdk needs accurate time to connect to the live network
        bindings.Bindings.setTimeSource(timeSource(context))

        appendLogsTo(findViewById<TextView>(R.id.ConsoleView), this)

        // Load the network definition file from a file hosted on
        // an s3 bucket, and validate it with a certificate stored in
        // res/raw/mainnet.crt
        val ndf = downloadNdf(this.applicationContext, NDF_URL_MAIN)

        // The statePath is a directory that holds cMix xx network state
        val basePath = context.filesDir.toString()
        val statePath = Path.of(basePath, "state").toString()
        Log.println(Log.INFO, "xxdk", "XXDK STATE PATH: $statePath")
        val stateFile = File(statePath)

        // Instantiate a user with the state directory password "Hello"
        val secret = "Hello".toByteArray();
        val cMixParamsJSON = "".toByteArray();
        if (!stateFile.exists()) {
            bindings.Bindings.newCmix(ndf, statePath, secret, "")
        }
        net = bindings.Bindings.loadCmix(statePath, secret, cMixParamsJSON)

        val receptionID = Base64.getEncoder().encodeToString(net.receptionID)
        Log.println(Log.INFO, "xxdk",
            "cMix Reception ID: $receptionID")

        var dmID: ByteArray
        try {
            dmID = net.ekvGet("MyDMID")
        } catch (e: Exception) {
            Log.println(Log.INFO, "xxdk",
                "Generating DM Identity...")
            // NOTE: This will be deprecated in favor of generateCodenameIdentity(...)
            dmID = bindings.Bindings.generateChannelIdentity(net.id);
            Log.println(Log.INFO, "xxdk", "Exported Codename Blob: " +
                    Base64.getEncoder().encodeToString(dmID))
            net.ekvSet("MyDMID", dmID)
        }


        Log.println(Log.INFO, "xxdk", "Exported Codename Blob: " +
                Base64.getEncoder().encodeToString(dmID))

        val notifications = bindings.Bindings.loadNotifications(net.id)

        val msgReceiveView = findViewById<TextView>(R.id.ReceivedMessagesView)
        val events = DmEvents(msgReceiveView)
        val receiver = DmReceiver(msgReceiveView)
        val receiverBuilder = DmReceiverBuilder(receiver)

        //Note: you can use `newDmManagerMobile` here instead if you want to work with
        //an SQLite database.
        // This interacts with the network and requires an accurate clock to connect or you'll see
        // "Timestamp of request must be within last 5 seconds." in the logs.
        // If you have trouble shutdown and start your emulator.
        DM = bindings.Bindings.newDMClient(net.id, notifications.id,
            dmID, receiverBuilder, events)

        Log.println(Log.INFO, "xxdk", "DMPUBKEY: " +
                "${Base64.getEncoder().encodeToString(DM.publicKey)}")
        Log.println(Log.INFO, "xxdk", "DMTOKEN: ${DM.token}")
        net.startNetworkFollower(5000)
        net.waitForNetwork(30000)

        val SendButton = findViewById<TextView>(R.id.SendButton)
        val SendTextInput = findViewById<TextView>(R.id.SendTextInput)
        SendButton.setOnClickListener {
            val text = SendTextInput.editableText.toString()
            DM.sendText(DM.publicKey, DM.token.toInt(), text, 0, cMixParamsJSON)
            SendTextInput.editableText.clear()
        }
    }
}