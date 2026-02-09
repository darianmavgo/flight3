Help me figure out some clean capable design choices.

I want Sqliter to stay a pure flutter app. 
I want FLight3 to stay a pure go/pocketbase app. 

I want Sqliter in Desktop mode to handle all local file browsing. But it would also be really smart when user requests a non sqlite file to forward the local file request to Flight3 and have it do conversion and return the sqlite file to Sqliter.

I also want to come up a much cleaner more useful start view for Sqliter. Some view that contains all the datasets that have already been downloaded and converted to sqlite. The view could also be credentials recently used and local files that have been viewed recently. Give me ideas what else we should show in this view.

Give me ideas on how to make the pagination more obvious and easier to use.  If TrinaGrid has other features that we aren't using that we should, let me know.. 