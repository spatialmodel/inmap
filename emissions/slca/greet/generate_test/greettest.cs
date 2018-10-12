using System.Reflection;
using System.Windows.Forms;
using Greet.DataStructureV4.Interfaces;
using Greet.Model.Interfaces;
using System;
using System.Collections.Generic;
using System.Linq;
using Greet.Model;
using Greet.DataStructureV4.Entities;

namespace GenerateTest
{
    /// <summary>
    /// This plugin example shows a very simple way to grab results from the GREET calculated pathways
    /// </summary>
    internal class ResultsAccess : APlugin
    {
        /// <summary>
        /// Controller that allows access to the data and functions
        /// </summary>
        public static IGREETController controler;

        /// <summary>
        /// A array of the menu items for this plugin
        /// </summary>
        ToolStripMenuItem[] items = new ToolStripMenuItem[1];

        #region APlugin
        /// <summary>
        /// Initialize the plugin, called once after the DLL is loaded into GREET
        /// </summary>
        /// <param name="controler"></param>
        /// <returns></returns>
        public override bool InitializePlugin(IGREETController controler)
        {
            //init the controller that is used to send action and data requests to GREET
            ResultsAccess.controler = controler;

            //init menu items collection for this example
            ToolStripMenuItem ex = new ToolStripMenuItem("Generate Test Data");
            ex.Click += (s, e) =>
            {
                Run();
            };
            this.items[0] = ex;

            return true;
        }

        public override string GetPluginName
        {
            get { return "API Example1 : Retriving Results"; }
        }

        public override string GetPluginDescription
        {
            get { return "Allows the user to select pathways and mixes, displays the results of the selected item"; }
        }

        public override string GetPluginVersion
        {
            get { return Assembly.GetExecutingAssembly().GetName().Version.ToString(); }
        }

        public override System.Drawing.Image GetPluginIcon
        {
            get { return null; }
        }
        #endregion

        #region menu items

        /// <summary>
        /// Called when the GREET main form is initializing, returns the menu items for this plugin
        /// </summary>
        /// <returns></returns>
        public override System.Windows.Forms.ToolStripMenuItem[] GetMainMenuItems()
        {
            return this.items;
        }

        /// <summary>
        /// Get information about all the pathways and mixes.
        /// </summary>
        public void Run()
        {
            //Gets the dictionary of IResource object indexed by IResource.Id
            IGDataDictionary<int, IResource> resources = ResultsAccess.controler.CurrentProject.Data.Resources;
            //Gets the dictionary of IPathways object indexed by IPathway.Id
            IGDataDictionary<int, IPathway> pathways = ResultsAccess.controler.CurrentProject.Data.Pathways;
            //Gets the dictionary of IMixes object indexed by IMid.Id
            IGDataDictionary<int, IMix> mixes = ResultsAccess.controler.CurrentProject.Data.Mixes;
            // Gets the dictionary of vehicles.
            IGDataDictionary<int, IVehicle> vehicles = ResultsAccess.controler.CurrentProject.Data.Vehicles;

            int i = 0;

            using (System.IO.StreamWriter file = new System.IO.StreamWriter(@"datafor_test.json"))
            {
                file.Write("{\"greetTests\":[");

                // resource name = resource.Name
                foreach (IPathway pathway in pathways.AllValues)
                {
                    IResults result = null;
                    int productID = -1;
                    string pathwayName = "";

                    //We ask the pathway what is the product defined as the main product for this pathway
                    //then store an integer that corresponds to an IResource.ID
                    productID = ResultsAccess.controler.CurrentProject.Data.Helper.PathwayMainOutputResouce(pathway.Id);
                    //We use the ID of the Resource that corresponds to the main output of the pathway to get the correct results
                    try
                    {
                        Dictionary<IIO, IResults> availableResults = pathway.GetUpstreamResults(ResultsAccess.controler.CurrentProject.Data);

                        Guid desiredOutput = new Guid();
                        foreach (IIO io in availableResults.Keys.Where(item => item.ResourceId == productID))
                        {
                            desiredOutput = io.Id;
                            if (io.Id == pathway.MainOutput)
                            {
                                desiredOutput = io.Id;
                                break;
                            }
                        }
                        result = availableResults.SingleOrDefault(item => item.Key.Id == desiredOutput).Value;
                        //We set the string variable as the name of the pathway
                        pathwayName = pathway.Name;

                        //if we found a pathway or a mix and we have all the necessary parameters 
                        //we Invoke the SetResults method of our user control in charge of displaying the life cycle upstream results
                        if (result != null && productID != -1 && !String.IsNullOrEmpty(pathwayName))
                        {
                            System.Guid id = new Guid("dddddddddddddddddddddddddddddddd");
                            SetResults(file, pathwayName, "", id, result, i);
                            i++;
                        }


                        //Greet.DataStructureV4.Entities.Vehicle v = vehicle as Greet.DataStructureV4.Entities.Vehicle;
                        
                        //getting the results for each individual vertex (canonical process representation of a process model) in the pathway
                        Greet.DataStructureV4.Entities.Pathway p = pathway as Greet.DataStructureV4.Entities.Pathway;
                        foreach (KeyValuePair<Guid, Greet.DataStructureV4.ResultsStorage.CanonicalProcess> pair in p.CanonicalProcesses)
                        {//iterate over all the processes in the pathway
                            Guid vertexUniqueId = pair.Key;
                            Greet.DataStructureV4.ResultsStorage.CanonicalProcess vertexProcessRepresentation = pair.Value;

                            int processModelId = pair.Value.ModelId;
                            string processName = ResultsAccess.controler.CurrentProject.Data.Processes.ValueForKey(processModelId).Name;

                            foreach (KeyValuePair<Guid, Greet.DataStructureV4.ResultsStorage.CanonicalOutput> outputPair in vertexProcessRepresentation.OutputsResults)
                            {//iterating over all the allocated outputs that have upstream results associated with them

                                Guid outputUniqueGui = outputPair.Key;
                                Greet.DataStructureV4.ResultsStorage.CanonicalOutput output = outputPair.Value;
                                IResults outputUpstreamResults = output.Results;

                                //if we found a pathway or a mix and we have all the necessary parameters 
                                //we Invoke the SetResults method of our user control in charge of displaying the life cycle upstream results
                                if (outputUpstreamResults != null && productID != -1 && !String.IsNullOrEmpty(pathwayName))
                                {
                                    SetResults(file, pathwayName, processName, outputUniqueGui, outputUpstreamResults, i);
                                    i++;
                                }
                                
                                AOutput referenceToOriginalProcessOutputInstance = outputPair.Value.Output;
                                double calculatedOutputBiogenicCarbonMassRatio = outputPair.Value.MassBiogenicCarbonRatio;
                            }
                        }
                    }
                    catch (System.Exception e)
                    {
                        Console.WriteLine("Problem with pathway {1}: {0}", e, pathway.Name);
                    }
                }

                foreach (IMix mix in mixes.AllValues)
                {
                    IResults result = null;
                    int productID = -1;
                    string name = "";

                    //We ask the mix what is the product defined as the main product for this mix
                    //then store an integer that corresponds to an IResource.ID
                    productID = mix.MainOutputResourceID;
                    //We use the ID of the Resource that corresponds to the main output of the pathway to get the correct results
                    var upstream = mix.GetUpstreamResults(ResultsAccess.controler.CurrentProject.Data);

                    if (null == upstream.Keys.SingleOrDefault(item => item.ResourceId == productID))
                    {
                        MessageBox.Show("Selected mix does not produce the fuel selected. Please remove it from the Fule Types list");
                        return;
                    }

                    //a mix has a single output so we can safely do the folowing
                    result = upstream.SingleOrDefault(item => item.Key.ResourceId == productID).Value;

                    //We set the string variable as the name of the pathway
                    name = mix.Name;

                    //if we found a pathway or a mix and we have all the necessary parameters 
                    //we Invoke the SetResults method of our user control in charge of displaying the life cycle upstream results
                    if (result != null && productID != -1 && !String.IsNullOrEmpty(name))
                    {
                        System.Guid id = new Guid("dddddddddddddddddddddddddddddddd");
                        SetResults(file, name, "", id, result, i);
                        i++;
                    }
                }
                file.WriteLine("\n]}");
            }
        }
        /// <summary>
        /// Invoked when a pathway is selected in order to represent the life cycle results
        /// for the product produced by this pathway and defined as it's main output (which is
        /// equivalent to the main output of the last process in the pathway)
        /// </summary>
        /// <param name="name">Name of the pathway, will simply be displayed as is</param>
        /// <param name="results">Result object from the pathway for the desired productID</param>
        /// <returns>Returns 0 if succeed</returns>
        public int SetResults(System.IO.StreamWriter file, string pathwayName, string processName, Guid outputID, IResults results, int i)
        {
            if (i != 0)
                file.Write(",\n");
            else
                file.Write("\n");

            //Check that the resuls object is non null
            if (results == null)
                return -1;

            file.WriteLine("\t{");
            file.WriteLine("\t\t\"i\": \"{0}\",", i);
            file.WriteLine("\t\t\"Pathway\": \"{0}\",", pathwayName);
            file.WriteLine("\t\t\"Process\": \"{0}\",", processName);
            file.WriteLine("\t\t\"OutputID\": \"{0}\",", outputID);

            //Get an instance of the data object that we are going to use to look for a Resource
            IData data = ResultsAccess.controler.CurrentProject.Data;
            //Gets the dictionary of IGases object indexed by IGas.Id
            IGDataDictionary<int, IGas> gases = ResultsAccess.controler.CurrentProject.Data.Gases;
            IGDataDictionary<int, IResource> resources = ResultsAccess.controler.CurrentProject.Data.Resources;

            file.Write("\t\t\"WTPEmis\": {");
            int ii = 0;
            foreach (KeyValuePair<int, IValue> emission in results.WellToProductEmissions())
            {
                if (ii != 0)
                    file.Write(",\n");
                else
                    file.Write("\n");
                IGas gas = gases.ValueForKey(emission.Key);
                //Format the value nicely using the quantity and the unit as well as the preferences defined by the user in the main UI GREET preferences
                file.Write("\t\t\t\"{0}\": {{\"val\":{1},\"units\":\"{2}\"}}", gas.Name, emission.Value.Value, emission.Value.UnitExpression);
                ii++;
            }

            file.Write("\n\t\t},\n\t\t\"OnSiteEmis\": {");
            ii = 0;
            foreach (KeyValuePair<int, IValue> emission in results.OnSiteEmissions())
            {
                if (ii != 0)
                    file.Write(",\n");
                else
                    file.Write("\n");
                IGas gas = gases.ValueForKey(emission.Key);
                //Format the value nicely using the quantity and the unit as well as the preferences defined by the user in the main UI GREET preferences
                file.Write("\t\t\t\"{0}\": {{\"val\":{1},\"units\":\"{2}\"}}", gas.Name, emission.Value.Value, emission.Value.UnitExpression);
                ii++;
            }

            file.Write("\n\t\t},\n\t\t\"WTPResources\": {");
            ii = 0;
            foreach (KeyValuePair<int, IValue> resUse in results.WellToProductResources())
            {
                if (ii != 0)
                    file.Write(",\n");
                else
                    file.Write("\n");
                IResource res = resources.ValueForKey(resUse.Key);
                //Format the value nicely using the quantity and the unit as well as the preferences defined by the user in the main UI GREET preferences
                file.Write("\t\t\t\"{0}\": {{\"val\":{1},\"units\":\"{2}\"}}", res.Name, resUse.Value.Value, resUse.Value.UnitExpression);
                ii++;
            }

            file.Write("\n\t\t},\n\t\t\"OnSiteResources\": {");
            ii = 0;
            foreach (KeyValuePair<int, IValue> resUse in results.OnSiteResources())
            {
                if (ii != 0)
                    file.Write(",\n");
                else
                    file.Write("\n");
                IResource res = resources.ValueForKey(resUse.Key);
                //Format the value nicely using the quantity and the unit as well as the preferences defined by the user in the main UI GREET preferences
                file.Write("\t\t\t\"{0}\": {{\"val\":{1},\"units\":\"{2}\"}}", res.Name, resUse.Value.Value, resUse.Value.UnitExpression);
                ii++;
            }
            file.Write("\n\t\t},\n");

            //Displays the functional unit for this results, very important in order to know if we are looking at results
            //per joule of product, or per cubic meters of product, or per kilograms of prododuct
            file.Write("\t\t\"FunctionalUnit\": \"{0}\"\n\t}}", results.FunctionalUnit);
            //If the user wants to see results in a different functional unit, the IValue quantity must be converted to the desired functional unit

            return 0;
        }

        #endregion
    }
}
