package network.xx.xxdkexample

import android.text.method.ScrollingMovementMethod
import android.widget.TextView
import androidx.lifecycle.LifecycleOwner
import androidx.lifecycle.Observer
import androidx.lifecycle.ViewModel
import androidx.lifecycle.liveData
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.Dispatchers


// LogViewer streams the logs from the logcat command to an observer:
//      val logViewer by viewModels<LogViewer>()
//      logViewer.logBuffer().observe(this, Observer {
//          m -> logMessageTextView.append("$m\n")
//      })
class LogViewer : ViewModel() {
    fun logBuffer() = liveData(viewModelScope.coroutineContext + Dispatchers.IO) {
        // clear the logs
        //Runtime.getRuntime().exec("logcat -c")
        // now emit each line of the logs from here on out
        val logcat = Runtime.getRuntime().exec("logcat")
        logcat.inputStream.bufferedReader().useLines {
            lines -> lines.forEach {
                l -> emit(l)
            }
        }
    }
}

// appendLogsTo is a helper function to append LogViewer logs to a
// TextView object.
fun appendLogsTo(t: TextView, owner: LifecycleOwner) {
    t.movementMethod = ScrollingMovementMethod()
    val logViewer = LogViewer()

    logViewer.logBuffer().observe(owner, Observer {
            m: String -> t.append("$m\n")
            val scrollAmount = t.layout.getLineTop(t.lineCount) - t.height;
            // if there is no need to scroll, scrollAmount will be <=0
            if (scrollAmount > 0)
                t.scrollTo(0, scrollAmount);
    })
}